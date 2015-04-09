package dir

import (
	"crypto/sha512"
	"encoding/base64"
	"hash"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/lib/fshash"
	"polydawn.net/repeatr/output"
)

const Type = "dir"

var _ output.Output = &Output{} // interface assertion

type Output struct {
	spec          def.Output
	hasherFactory func() hash.Hash
}

func New(spec def.Output) output.Output {
	if spec.Type != Type {
		panic(errors.ProgrammerError.New("This output implementation supports definitions of type %q, not %q", Type, spec.Type))
	}
	return &Output{
		spec:          spec,
		hasherFactory: sha512.New384,
	}
}

func (o Output) Apply(basePath string) <-chan output.Report {
	done := make(chan output.Report)
	go func() {
		defer close(done)
		try.Do(func() {
			// walk filesystem, copying and accumulating data for integrity check
			bucket := &fshash.MemoryBucket{}
			err := fshash.FillBucket(basePath, o.spec.URI, bucket, o.hasherFactory)
			if err != nil {
				panic(err) // TODO this is not well typed, and does not clearly indicate whether scanning or committing had the problem
			}

			// hash whole tree
			actualTreeHash, _ := fshash.Hash(bucket, o.hasherFactory)

			// report
			o.spec.Hash = base64.URLEncoding.EncodeToString(actualTreeHash)
			done <- output.Report{nil, o.spec}
		}).Catch(output.Error, func(err *errors.Error) {
			done <- output.Report{err, o.spec}
		}).CatchAll(func(err error) {
			// All errors we emit will be under `output.Error`'s type.
			done <- output.Report{output.UnknownError.Wrap(err).(*errors.Error), o.spec}
		}).Done()
	}()
	return done
}
