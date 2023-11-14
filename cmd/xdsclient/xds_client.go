package xdsclient

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/trinhdaiphuc/grpc-xds-example/pkg/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	_ "google.golang.org/grpc/xds" // To install the xds resolvers and balancers.
)

var (
	concurrency int
	duration    string
	target      string
)

func RegisterCommand(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "xds-client",
		Short: "Run the client service",
		Run: func(cmd *cobra.Command, args []string) {
			run(args)
		},
	}
	cmd.Flags().IntVarP(&concurrency, "concurrency", "c", 5, "Number of concurrent requests to run the client service")
	cmd.Flags().StringVarP(&duration, "duration", "d", "5s", "A duration string is a possibly signed sequence of decimal numbers, each with optional fraction and a unit suffix, "+
		"such as \"300ms\", \"-1.5h\" or \"2h45m\"."+
		" Valid time units are \"ns\", \"us\" (or \"µs\"), \"ms\", \"s\", \"m\", \"h\".")
	cmd.Flags().StringVarP(&target, "target", "t", "xds:///localhost:50051", "uri of the server")
	parent.AddCommand(cmd)
}

func run(_ []string) {
	if !strings.HasPrefix(target, "xds:///") {
		log.Fatalf("-target must use a URI with scheme set to 'xds'")
	}

	// Make another ClientConn with xds policy.
	xdsConn, err := grpc.Dial(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second, // client ping server if no activity for this long
			Timeout:             20 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer xdsConn.Close()

	go client.LoadTest(xdsConn, duration, concurrency)

	client.ServeHTTP()
}
