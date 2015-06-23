package tests

import (
	"fmt"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/testutil"
	"polydawn.net/repeatr/testutil/filefixture"
)

/*
	Checks round-trip hash consistency for the input and output halves of a transmat system.

	- Creates a fixture filesystem
	- Scans it with the output system
	- Places it in a new filesystem with the input system and the scanned hash
	- Checks the new filesystem matches the original
*/
func CheckRoundTrip(kind integrity.TransmatKind, transmatFabFn integrity.TransmatFactory, bounceURI string, addtnlDesc ...string) {
	Convey("SPEC: Round-trip scanning and remaking a filesystem should agree on hash and content"+testutil.AdditionalDescription(addtnlDesc...), testutil.Requires(
		testutil.RequiresRoot,
		func() {
			transmat := transmatFabFn("./workdir")
			for _, fixture := range filefixture.All {
				Convey(fmt.Sprintf("- Fixture %q", fixture.Name), FailureContinues, func() {
					uris := []integrity.SiloURI{integrity.SiloURI(bounceURI)}
					// setup fixture
					fixture.Create("./fixture")
					// scan it with the transmat
					dataHash := transmat.Scan(kind, "./fixture", uris)
					// materialize what we just scanned (along the way, requires hash match)
					arena := transmat.Materialize(kind, dataHash, uris, integrity.AcceptHashMismatch)
					// assert hash match
					// (normally survival would attest this, but we used the `AcceptHashMismatch` to supress panics in the name of letting the test see more after failures.)
					So(arena.Hash(), ShouldEqual, dataHash)
					// check filesystem to match original fixture
					// (do this check even if the input raised a hash mismatch, because it can help show why)
					rescan := filefixture.Scan(arena.Path())
					comparisonLevel := filefixture.CompareDefaults &^ filefixture.CompareSubsecond
					So(rescan.Describe(comparisonLevel), ShouldEqual, fixture.Describe(comparisonLevel))
				})
			}
		},
	))
}