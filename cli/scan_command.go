package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/codegangsta/cli"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor/util"
	"polydawn.net/repeatr/io"
)

func ScanCommandPattern(output io.Writer) cli.Command {
	return cli.Command{
		Name:  "scan",
		Usage: "Scan a local filesystem, optionally storing the data into a silo",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "kind",
				Usage: "What kind of data storage format to work with.",
			},
			cli.StringFlag{
				Name:  "path",
				Value: ".",
				Usage: "Optional.  The local filesystem path to scan.  Defaults to your current directory.",
			},
			cli.StringFlag{
				Name:  "silo",
				Usage: "Optional.  A Silo URI to upload data to.",
			},
		},
		Action: func(ctx *cli.Context) {
			// args parse
			var warehouses []string
			if ctx.IsSet("silo") {
				warehouses = []string{ctx.String("silo")}
			}
			outputSpec := def.Output{
				Type:       ctx.String("kind"),
				Warehouses: warehouses,
				// Filters: might want
				MountPath: ctx.String("path"),
			}
			if outputSpec.Type == "" {
				panic(Error.NewWith("Missing argument: \"kind\" is a required parameter for scan", SetExitCode(EXIT_BADARGS)))
			}
			if outputSpec.MountPath == "" {
				outputSpec.MountPath = "."
			}
			// invoke
			outputResult := Scan(outputSpec)
			// output
			// FIXME serialization format.
			//  should be especially pretty and human-friendly; deserves custom code.
			//    really, you want that anyway for things like hassle-free syntax in practice for single URIs without an array, etc.
			msg, err := json.Marshal(outputResult)
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(output, "%s\n", string(msg))
		},
	}
}

/*
	Spits out a chunk of json on stdout that can be used as
	a `Input` specification in a `Formula`.
*/
func Scan(outputSpec def.Output) def.Output {
	// TODO validate MountPath exists, give nice errors

	// todo: create validity checking api for URIs, check them all before launching anything
	warehouses := make([]integrity.SiloURI, len(outputSpec.Warehouses))
	for i, wh := range outputSpec.Warehouses {
		warehouses[i] = integrity.SiloURI(wh)
	}

	// So, this CLI command is *not* in its rights to change the subject area,
	//  so take that as a pretty strong hint that filters are going to have to pass down *into* transmats.
	commitID := util.DefaultTransmat().Scan(
		// All of this stuff that's type-coercing?
		//  Yeah these are hints that this stuff should be facing data validation.
		integrity.TransmatKind(outputSpec.Type),
		outputSpec.MountPath,
		warehouses,
	)

	return def.Output{
		Type:       outputSpec.Type,
		Warehouses: outputSpec.Warehouses,
		Hash:       string(commitID),
	}
}
