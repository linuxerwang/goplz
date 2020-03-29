package debug

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli"

	"github.com/linuxerwang/goplz/vfs"
)

// DebugCmd is for subcommand "init".
var DebugCmd = &cli.Command{
	Name:  "debug",
	Usage: "debug goplz",
	Action: func(ctx *cli.Context) error {
		fmt.Println("Debug goplz: ", ctx.Args().First())
		return nil
	},
}

func ondemandDebug(fs vfs.FileSystem) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR1)
	go func() {
		for {
			<-c
			ioutil.WriteFile(fmt.Sprintf("/tmp/goplz-%d.log", os.Getpid()), []byte(fs.String()), os.ModePerm)
		}
	}()
}
