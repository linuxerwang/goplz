package main

import (
	"log"
	"os"

	cli "github.com/urfave/cli/v2"

	"github.com/linuxerwang/goplz/commands/debug"
	initialize "github.com/linuxerwang/goplz/commands/init"
	"github.com/linuxerwang/goplz/commands/start"
	"github.com/linuxerwang/goplz/commands/stop"
	"github.com/linuxerwang/goplz/commands/version"
)

func main() {
	app := &cli.App{
		Name:  "goplz",
		Usage: "A fuse mount tool for Please to better support Golang development",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "verbose",
				Value: false,
				Usage: "print verbose logs",
			},
		},
		Commands: []*cli.Command{
			debug.DebugCmd,
			initialize.InitCmd,
			start.StartCmd,
			stop.StopCmd,
			version.VersionCmd,
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

