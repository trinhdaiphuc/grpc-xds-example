package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/trinhdaiphuc/grpc-xds-example/cmd/client"
	"github.com/trinhdaiphuc/grpc-xds-example/cmd/server"
	"github.com/trinhdaiphuc/grpc-xds-example/cmd/xds"
	"github.com/trinhdaiphuc/grpc-xds-example/cmd/xdsclient"
	"os"
)

func newRootCmd() (cmd *cobra.Command) {
	cmd = &cobra.Command{
		Use: "grpc-example",
	}

	client.RegisterCommand(cmd)
	server.RegisterCommand(cmd)
	xdsclient.RegisterCommand(cmd)
	xds.RegisterCommand(cmd)
	return cmd
}

func main() {
	cmd := newRootCmd()

	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}
