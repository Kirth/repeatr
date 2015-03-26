package fshash

import (
	"archive/tar"
	"hash"

	"github.com/spacemonkeygo/errors"
	"github.com/ugorji/go/codec"
	"polydawn.net/repeatr/lib/treewalk"
)

/*
	Walks the tree of files and metadata arrayed in a `Bucket` and
	constructs a tree hash over them.  The root of the tree hash is returned.
	The returned root has can be said to verify the integrity of the
	entire tree (much like a Merkle tree).

	The serial structure is expressed something like the following:

		{"node": $dir.metadata.hash,
		 "leaves": [
			{"node": $file1.metadata.hash, "content": $file1.contentHash},
			{"node": $subdir.metadata.hash,
			 "leaves": [ ... ]},
		 ]
		}

	This expression is made in cbor (rfc7049) format with indefinite-length
	arrays and a fixed order for all map fields.  Every structure starting
	with "node" is itself hashed and that value substituted in before
	hashing the parent.  Since the metadata hash contains the file/dir name,
	and the tree itself is traversed in sorted order, the entire structure
	is computed deterministically and unambiguously.
*/
func Hash(bucket Bucket, hasherFactory func() hash.Hash) ([]byte, error) {
	// Hack around codec not exporting things very usefully -.-
	const magic_RAW = 0
	const magic_UTF8 = 1
	// At every point in the visitation, children need to submit their hashes back up the tree.
	// Prime the pump with a special reaction for when the root returns; every directory preVisit attaches hoppers for children thereon.
	upsubs := make(upsubStack, 0)
	var finalAnswer []byte
	upsubs.Push(func(x []byte) {
		finalAnswer = x
	})
	// Also keep a stack of hashers in use because they jump across the pre/post visit gap.
	hashers := make(hasherStack, 0)
	// Visitor definitions
	preVisit := func(node treewalk.Node) error {
		record := node.(RecordIterator).Record()
		hasher := hasherFactory()
		_, enc := codec.GenHelperEncoder(codec.NewEncoder(hasher, new(codec.CborHandle)))
		enc.EncodeMapStart(2) // either way it's header + one of leaves or contenthash
		enc.EncodeString(magic_UTF8, "m")
		record.Metadata.Marshal(hasher)
		if record.Metadata.Typeflag == tar.TypeDir {
			// open the "leaves" array
			// this may end up being an empty dir, but we act the same regardless
			// (and we don't have that information here since the iterator has tunnel vision)
			enc.EncodeString(magic_UTF8, "l")
			hasher.Write([]byte{codec.CborStreamArray})
			upsubs.Push(func(x []byte) {
				enc.EncodeStringBytes(magic_RAW, x)
			})
			hashers.Push(hasher)
		} else {
			// heap the object's content hash in
			enc.EncodeString(magic_UTF8, "h")
			enc.EncodeStringBytes(magic_RAW, record.ContentHash)
			// finalize our hash here and upsub to save us the work of hanging onto the hasher until the postvisit call
			upsubs.Peek()(hasher.Sum(nil))
		}
		return nil
	}
	postVisit := func(node treewalk.Node) error {
		record := node.(RecordIterator).Record()
		if record.Metadata.Typeflag == tar.TypeDir {
			hasher := hashers.Pop()
			// close off the "leaves" array
			// No map-close necessary because we used a fixed length map.
			hasher.Write([]byte{0xff}) // should be `codec.CborStreamBreak` but upstream has an export bug :/
			// pop out this dir's hoppers for children data
			upsubs.Pop()
			// hash and upsub
			upsubs.Peek()(hasher.Sum(nil))

		}
		return nil
	}
	// Traverse
	if err := treewalk.Walk(bucket.Iterator(), preVisit, postVisit); err != nil {
		panic(err) // none of our code has known believable error returns.
	}
	// Sanity check no node left behind
	_ = upsubs.Pop()
	if !upsubs.Empty() || !hashers.Empty() {
		panic(errors.ProgrammerError.New("invariant failed after bucket records walk"))
	}
	// return the result upsubbed by the root
	return finalAnswer, nil
}

type upsubStack []func([]byte)

func (s upsubStack) Empty() bool          { return len(s) == 0 }
func (s upsubStack) Peek() func([]byte)   { return s[len(s)-1] }
func (s *upsubStack) Push(x func([]byte)) { (*s) = append((*s), x) }
func (s *upsubStack) Pop() func([]byte) {
	x := (*s)[len(*s)-1]
	(*s) = (*s)[:len(*s)-1]
	return x
}

// look me in the eye and tell me again how generics are a bad idea
type hasherStack []hash.Hash

func (s hasherStack) Empty() bool       { return len(s) == 0 }
func (s hasherStack) Peek() hash.Hash   { return s[len(s)-1] }
func (s *hasherStack) Push(x hash.Hash) { (*s) = append((*s), x) }
func (s *hasherStack) Pop() hash.Hash {
	x := (*s)[len(*s)-1]
	(*s) = (*s)[:len(*s)-1]
	return x
}