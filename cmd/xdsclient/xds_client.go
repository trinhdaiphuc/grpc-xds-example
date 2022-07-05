package xdsclient

import (
	"log"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"grpc-loadbalancing/pkg/client"

	_ "google.golang.org/grpc/xds" // To install the xds resolvers and balancers.
)

var (
	concurrency       int
	duration          string
	target            string
	grpcClientCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "grpc_xds_client_request_total",
		Help:        "Number of total request.",
		ConstLabels: prometheus.Labels{"version": "1"},
	})
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
	prometheus.MustRegister(grpcClientCounter)
	parent.AddCommand(cmd)
}

func run(_ []string) {
	if !strings.HasPrefix(target, "xds:///") {
		log.Fatalf("-target must use a URI with scheme set to 'xds'")
	}

	creeds := insecure.NewCredentials()
	//creds, err := xds.NewClientCredentials(xds.ClientOptions{
	//	FallbackCreds: insecure.NewCredentials()
	//})
	// Make another ClientConn with xds policy.
	xdsConn, err := grpc.Dial(
		target,
		grpc.WithTransportCredentials(creeds),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer xdsConn.Close()

	go client.LoadTest(xdsConn, duration, concurrency, grpcClientCounter)

	client.ServeHTTP()
}
