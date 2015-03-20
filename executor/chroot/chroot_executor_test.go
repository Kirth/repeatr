package chroot

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/testutil"
)

func Test(t *testing.T) {
	Convey("Given a rootfs input that errors", t,
		testutil.WithTmpdir(func() {
			formula := def.Formula{
				Inputs: []def.Input{
					{
						Type:     "tar",
						Location: "/",
						URI:      "/nonexistance/in/its/most/essential/unform.tar.gz",
					},
				},
			}
			executor := &Executor{
				workspacePath: "chroot_workspace",
			}
			So(os.Mkdir(executor.workspacePath, 0755), ShouldBeNil)

			Convey("We should get an InputError", func() {
				So(func() { executor.Run(formula) }, testutil.ShouldPanicWith, input.InputError)
			})
		}),
	)

	projPath, _ := os.Getwd()
	projPath = filepath.Dir(filepath.Dir(projPath))

	testutil.Convey_IfHaveRoot("Given a rootfs", t,
		testutil.WithTmpdir(func() {
			formula := def.Formula{
				Inputs: []def.Input{
					{
						Type:     "tar",
						Location: "/",
						URI:      filepath.Join(projPath, "assets/ubuntu.tar.gz"),
					},
				},
			}
			executor := &Executor{
				workspacePath: "chroot_workspace",
			}
			So(os.Mkdir(executor.workspacePath, 0755), ShouldBeNil)

			Convey("The executor should be able to invoke echo", func() {
				formula.Accents = def.Accents{
					Entrypoint: []string{"echo", "echococo"},
				}

				job, _ := executor.Run(formula)
				So(job, ShouldNotBeNil)
				So(job.ExitCode(), ShouldEqual, 0) // TODO: this waits... the test should still pass if the reader happens first
				msg, err := ioutil.ReadAll(job.OutputReader())
				So(err, ShouldBeNil)
				So(string(msg), ShouldEqual, "echococo\n")
			})

			Convey("The executor should be able to check exit codes", func() {
				formula.Accents = def.Accents{
					Entrypoint: []string{"sh", "-c", "exit 14"},
				}

				job, _ := executor.Run(formula)
				So(job, ShouldNotBeNil)
				So(job.ExitCode(), ShouldEqual, 14)
			})

			Convey("The executor should report command not found clearly", func() {
				formula.Accents = def.Accents{
					Entrypoint: []string{"not a command"},
				}

				So(func() { executor.Run(formula) }, testutil.ShouldPanicWith, NoSuchCommandError)
			})

			Convey("Given another input", func() {
				formula.Inputs = append(formula.Inputs, def.Input{
					Type:     "dir",
					Location: "/data/test",
					URI:      "testdata",
				})
				So(os.Mkdir("testdata", 0755), ShouldBeNil)
				So(ioutil.WriteFile("testdata/1", []byte{}, 0644), ShouldBeNil)
				So(ioutil.WriteFile("testdata/2", []byte{}, 0644), ShouldBeNil)
				So(ioutil.WriteFile("testdata/3", []byte{}, 0644), ShouldBeNil)
				// FIXME: ugh, you either need some fixtures up in here, or we need to implement a "scan" feature asap

				//	Convey("The executor should be able to see the mounted files", func() {
				//		formula.Accents = def.Accents{
				//			Entrypoint: []string{"ls", "/data/test"},
				//		}
				//
				//		job, _ := executor.Run(formula)
				//		So(job, ShouldNotBeNil)
				//		So(job.ExitCode(), ShouldEqual, 0)
				//		msg, err := ioutil.ReadAll(job.OutputReader())
				//		So(err, ShouldBeNil)
				//		So(string(msg), ShouldEqual, "1 2 3\n")
				//	})
			})
		}),
	)
}
