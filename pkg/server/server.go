package server

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	pb "google.golang.org/grpc/examples/features/proto/echo"
	"io"
	"net/http"
)

type EcServer struct {
	pb.UnimplementedEchoServer
}

func (s *EcServer) UnaryEcho(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	return &pb.EchoResponse{Message: req.Message}, nil
}

func (s *EcServer) BidirectionalStreamingEcho(stream pb.Echo_BidirectionalStreamingEchoServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err = stream.Send(&pb.EchoResponse{Message: "Reply to bidirectional streaming request: " + in.Message}); err != nil {
			return err
		}
	}
}

func ServeHTTP() {
	// Register Prometheus metrics handler.
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	// Start the health check endpoint and make sure not to block
	fmt.Println("Server running on localhost:8000")
	_ = http.ListenAndServe(":8000", nil)
}
