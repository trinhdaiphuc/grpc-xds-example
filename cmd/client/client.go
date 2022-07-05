package client

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"github.com/trinhdaiphuc/grpc-xds-example/pkg/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	concurrency       int
	duration          string
	target            string
	grpcClientCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "grpc_client_request_total",
		Help:        "Number of total request.",
		ConstLabels: prometheus.Labels{"version": "1"},
	})
)

func RegisterCommand(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "client",
		Short: "Run the client service",
		Run: func(cmd *cobra.Command, args []string) {
			run(args)
		},
	}
	cmd.Flags().IntVarP(&concurrency, "concurrency", "c", 5, "Number of concurrent requests to run the client service")
	cmd.Flags().StringVarP(&duration, "duration", "d", "5s", "A duration string is a possibly signed sequence of decimal numbers, each with optional fraction and a unit suffix, "+
		"such as \"300ms\", \"-1.5h\" or \"2h45m\"."+
		" Valid time units are \"ns\", \"us\" (or \"µs\"), \"ms\", \"s\", \"m\", \"h\".")
	cmd.Flags().StringVarP(&target, "target", "t", "localhost:50051", "uri of the server")
	prometheus.MustRegister(grpcClientCounter)
	parent.AddCommand(cmd)
}

func run(_ []string) {
	// Make another ClientConn with round_robin policy.
	roundRobinConn, err := grpc.Dial(
		target,
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`), // This sets the initial balancing policy.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer roundRobinConn.Close()

	go client.LoadTest(roundRobinConn, duration, concurrency, grpcClientCounter)

	client.ServeHTTP()
}
