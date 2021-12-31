package version

import (
	"fmt"

	cli "github.com/urfave/cli/v2"
)

var (
	// Version is the version of goplz.
	Version string = "DEVELOPMENT"
)

// VersionCmd is for subcommand "version".
var VersionCmd = &cli.Command{
	Name:  "version",
	Usage: "show goplz version",
	Action: func(ctx *cli.Context) error {
		fmt.Printf("Version: %s\n", Version)
		return nil
	},
}
