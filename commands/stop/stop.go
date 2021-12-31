package stop

import (
	"fmt"
	"syscall"

	"github.com/linuxerwang/goplz/conf"
	cli "github.com/urfave/cli/v2"
)

// StopCmd is for subcommand "stop".
var StopCmd = &cli.Command{
	Name:  "stop",
	Usage: "stop goplz",
	Action: func(ctx *cli.Context) error {
		cfg := conf.Cfg()
		p := cfg.GetExistingProcess()
		if p == nil {
			fmt.Println("No existing goplz process found for this workspace")
			return nil
		}
		fmt.Printf("Stopping existing goplz process for workspace %s.\n", cfg.Workspace)
		if err := p.Signal(syscall.SIGQUIT); err != nil {
			fmt.Printf("Failed to send SIGQUIT to process %d.\n", p.Pid)
			return err
		}
		return nil
	},
}
