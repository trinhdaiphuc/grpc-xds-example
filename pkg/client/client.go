package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	ecpb "google.golang.org/grpc/examples/features/proto/echo"
)

func ServeHTTP() {
	// Register Prometheus metrics handler.
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	// Start the health check endpoint and make sure not to block
	fmt.Println("Server running on localhost:8080")
	_ = http.ListenAndServe(":8080", nil)
}

func callUnaryEcho(c ecpb.EchoClient, message string, counter prometheus.Counter) {
	_, err := c.UnaryEcho(context.Background(), &ecpb.EchoRequest{Message: message})
	if err != nil {
		logrus.Errorf("could not saned request: %v", err)
	}
	counter.Inc()
}

func callBidirectionalStreamingEcho(c ecpb.EchoClient, message string, counter prometheus.Counter) {
	stream, err := c.BidirectionalStreamingEcho(context.Background())
	if err != nil {
		logrus.Errorf("could not saned request: %v", err)
	}
	if stream == nil {
		return
	}
	for {
		err = stream.Send(&ecpb.EchoRequest{Message: message})
		if err != nil {
			return
		}
	}
	counter.Inc()
}

func startWorker(client ecpb.EchoClient, closeChannels <-chan bool, counter prometheus.Counter) {
	stream, err := client.BidirectionalStreamingEcho(context.Background())
	if err != nil {
		logrus.Errorf("could not send request: %v", err)
	}

	if stream != nil {
		go func() {
			for {
				_, err = stream.Recv()
				if err == io.EOF {
					// read done.
					return
				}
				if err != nil {
					logrus.Errorf("Failed to receive a note : %v", err)
				}
			}
		}()
	}

	for {
		select {
		case <-closeChannels:
			if stream != nil {
				stream.CloseSend()
			}
			return
		default:
			if stream != nil {
				err = stream.Send(&ecpb.EchoRequest{Message: "streaming message"})
				if err != nil {
					logrus.Errorf("could not send streaming message: %v", err)
				}
			}
			callUnaryEcho(client, "this is examples/load_balancing", counter)
		}
	}
}

func LoadTest(cc *grpc.ClientConn, duration string, concurrency int, counter prometheus.Counter) {
	hwc := ecpb.NewEchoClient(cc)
	fmt.Println("--- calling helloworld.Greeter/SayHello with xds ---")
	timeDuration, err := time.ParseDuration(duration)
	if err != nil {
		logrus.Fatalf("could not parse duration: %v", err)
	}
	ticker := time.NewTicker(timeDuration)
	closeChannels := make(chan bool, concurrency)
	for i := 0; i < concurrency; i++ {
		go startWorker(hwc, closeChannels, counter)
	}

	<-ticker.C
	for i := 0; i < concurrency; i++ {
		closeChannels <- true
	}
	fmt.Println("done")
}
