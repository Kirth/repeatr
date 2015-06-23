package tar

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/polydawn/gosh"
	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/lib/fspatch"
	"polydawn.net/repeatr/testutil"
	"polydawn.net/repeatr/testutil/filefixture"
)

const ubuntuTarballHash = "uJRF46th6rYHt0zt_n3fcDuBfGFVPS6lzRZla5hv6iDoh5DVVzxUTMMzENfPoboL"

func TestTarInputCompat(t *testing.T) {
	projPath, _ := os.Getwd()
	projPath = filepath.Dir(filepath.Dir(filepath.Dir(projPath)))

	Convey("Unpacking tars should match exec untar", t,
		testutil.Requires(testutil.RequiresRoot, testutil.WithTmpdir(func() {
			checkEquivalence := func(hash, filename string, paveBase bool) {
				transmat := New("./workdir/tar")

				// apply it; hope it doesn't blow up
				arena := transmat.Materialize(
					integrity.TransmatKind("tar"),
					integrity.CommitID(hash),
					[]integrity.SiloURI{
						integrity.SiloURI("file://" + filename),
					},
				)
				defer arena.Teardown()

				// do a native untar; since we don't have an upfront fixture
				//  for this thing, we'll compare the two as filesystems.
				// this is not well isolated from the host; consider improving that a todo.
				os.Mkdir("./untar", 0755)
				tarProc := gosh.Gosh(
					"tar",
					"-xf", filename,
					"-C", "./untar",
					gosh.NullIO,
				).RunAndReport()
				So(tarProc.GetExitCode(), ShouldEqual, 0)
				// native untar may or may not have an opinion about the base dir, depending on how it was formed.
				// but our scans do, so, if the `paveBase` flag was set to warn us that the tar was missing an "./" entry, flatten that here.
				if paveBase {
					So(fspatch.LUtimesNano("./untar", def.Epochwhen, def.Epochwhen), ShouldBeNil)
				}

				// scan and compare
				scan1 := filefixture.Scan(arena.Path())
				scan2 := filefixture.Scan("./untar")
				// boy, that's entertaining though: gnu tar does all the same stuff,
				//  except it doesn't honor our nanosecond timings.
				// also exclude bodies because they're *big*.
				comparisonLevel := filefixture.CompareDefaults &^ filefixture.CompareSubsecond &^ filefixture.CompareBody
				So(scan1.Describe(comparisonLevel), ShouldEqual, scan2.Describe(comparisonLevel))
			}

			Convey("Given a fixture tarball complete with base dir", func() {
				checkEquivalence(
					"BX0jm4jRNCg1KMbZfv4zp7ZaShx9SUXKaDrO-Xy6mWIoWOCFP5VnDHDDR3nU4PrR",
					filepath.Join(projPath, "data/fixture/tar_withBase.tgz"),
					false,
				)
			})

			Convey("Given a fixture tarball lacking base dir", func() {
				checkEquivalence(
					"ZdV3xhCGWeJmsfeHpDF4nF9stwvdskYwcepKMcOf7a2ziax1YGjQvGTJjRWFkvG1",
					filepath.Join(projPath, "data/fixture/tar_sansBase.tgz"),
					true,
				)
			})

			Convey("Given a fixture tarball containing ubuntu",
				testutil.Requires(testutil.RequiresLongRun, func() {
					checkEquivalence(
						ubuntuTarballHash,
						filepath.Join(projPath, "assets/ubuntu.tar.gz"),
						true,
					)
				}),
			)
		})),
	)

	Convey("Bouncing unusual tars should match hash", t,
		// where really all "unusual" means is "valid tar, but not from our own cannonical output".
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func() {
				checkBounce := func(hash, filename string) {
					transmat := New("./workdir/tar")

					// apply it; hope it doesn't blow up
					arena := transmat.Materialize(
						integrity.TransmatKind("tar"),
						integrity.CommitID(hash),
						[]integrity.SiloURI{
							integrity.SiloURI("file://" + filename),
						},
					)
					defer arena.Teardown()

					// scan and compare
					commitID := transmat.Scan(integrity.TransmatKind("tar"), arena.Path(), nil)
					// debug: gosh.Sh("tar", "--utc", "-xOvf", filename)
					So(commitID, ShouldEqual, integrity.CommitID(hash))

				}

				Convey("Given a fixture tarball complete with base dir", func() {
					checkBounce(
						"BX0jm4jRNCg1KMbZfv4zp7ZaShx9SUXKaDrO-Xy6mWIoWOCFP5VnDHDDR3nU4PrR",
						filepath.Join(projPath, "data/fixture/tar_withBase.tgz"),
					)
				})

				Convey("Given a fixture tarball lacking base dir", func() {
					checkBounce(
						"ZdV3xhCGWeJmsfeHpDF4nF9stwvdskYwcepKMcOf7a2ziax1YGjQvGTJjRWFkvG1",
						filepath.Join(projPath, "data/fixture/tar_sansBase.tgz"),
					)
				})

				// this won't fly until we support hardlinks; the original asset uses them.
				//	Convey("Given a fixture tarball containing ubuntu",
				//		testutil.Requires(testutil.RequiresLongRun, func() {
				//			checkBounce(
				//				ubuntuTarballHash,
				//				filepath.Join(projPath, "assets/ubuntu.tar.gz"),
				//			)
				//		}),
				//	)
			}),
		),
	)
}