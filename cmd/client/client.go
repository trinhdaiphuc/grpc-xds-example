package client

import (
	"log"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/resolver"

	"github.com/trinhdaiphuc/grpc-xds-example/pkg/client"
)

var (
	concurrency int
	duration    string
	target      string
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
	parent.AddCommand(cmd)
}

func run(_ []string) {
	// Make another ClientConn with round_robin policy.
	optsRetry := []grpc_retry.CallOption{
		grpc_retry.WithBackoff(grpc_retry.BackoffExponential(50 * time.Millisecond)),
		grpc_retry.WithCodes(codes.Unavailable),
		grpc_retry.WithMax(1),
		grpc_retry.WithPerRetryTimeout(100 * time.Millisecond),
	}
	grpc_retry.Disable()
	roundRobinConn, err := grpc.Dial(
		target,
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`), // This sets the initial balancing policy.
		grpc.WithResolvers(resolver.Get("dns")),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithReturnConnectionError(),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		grpc.WithChainUnaryInterceptor(
			grpc_prometheus.UnaryClientInterceptor,
			grpc_retry.UnaryClientInterceptor(optsRetry...),
		),
		grpc.WithChainStreamInterceptor(
			grpc_prometheus.StreamClientInterceptor,
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second, // client ping server if no activity for this long
			Timeout:             time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithConnectParams(
			grpc.ConnectParams{
				Backoff: backoff.Config{
					BaseDelay:  time.Second,
					Multiplier: 2,
					Jitter:     0.2,
					MaxDelay:   120 * time.Second,
				},
				MinConnectTimeout: 15 * time.Second,
			},
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer roundRobinConn.Close()

	go client.LoadTest(roundRobinConn, duration, concurrency)

	client.ServeHTTP()
}
