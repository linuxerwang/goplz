package initialize

import (
	"fmt"

	"github.com/linuxerwang/goplz/conf"
	cli "github.com/urfave/cli/v2"
)

// InitCmd is for subcommand "init".
var InitCmd = &cli.Command{
	Name:  "init",
	Usage: "initialize goplz",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "virtual_go_path",
			Value: "",
			Usage: "The virtual GOPATH to fuse mount. If not set, will use '.<top-folder>-gopath'. Virtual GOPATH Should not be anywhere of the top folder.",
		},
	},
	Action: func(ctx *cli.Context) error {
		if conf.HasGoplzRc() {
			fmt.Println("Current working directory has already been initialized.")
			return nil
		}

		conf.CreateGoplzRc(ctx.String("virtual_go_path"))

		fmt.Println("Now you can run `goplz start`.")
		return nil
	},
}
