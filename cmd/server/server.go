package server

import (
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/features/proto/echo"
	"grpc-loadbalancing/pkg/server"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

var address string

func RegisterCommand(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run the server service",
		Run: func(cmd *cobra.Command, args []string) {
			run(args)
		},
	}

	cmd.Flags().StringVarP(&address, "address", "a", ":50051", "server address to serve on")
	parent.AddCommand(cmd)
}

func run(_ []string) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_prometheus.UnaryServerInterceptor,
		)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_prometheus.StreamServerInterceptor,
		)),
	)

	// Echo server
	pb.RegisterEchoServer(s, &server.EcServer{})

	log.Printf("serving on %s\n", address)
	go func() {
		if err = s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	go server.ServeHTTP()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM)
	<-osSignal
}
