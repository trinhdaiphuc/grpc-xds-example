package xds

import (
	"context"
	"flag"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	httpserver "grpc-loadbalancing/pkg/server"
	"grpc-loadbalancing/pkg/xds"
	"k8s.io/client-go/kubernetes"
	"time"
)

var (
	port      uint
	nodeID    string
	namespace string
	zone      string
	region    string
	debug     bool
	services  []string
)

func RegisterCommand(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "xds",
		Short: "Run the xds service",
		Run: func(cmd *cobra.Command, args []string) {
			run(args)
		},
	}

	cmd.Flags().UintVarP(&port, "port", "p", 18000, "xDS management server port")
	cmd.Flags().StringVarP(&nodeID, "node-id", "i", "test-id", "Node ID")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace")
	cmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable xDS server debug logging")
	cmd.Flags().StringArrayVarP(&services, "services", "s", []string{"grpc-server"}, "grpc server service name")
	cmd.Flags().StringVarP(&zone, "zone", "z", "ap-southeast-1a", "Zone name")
	cmd.Flags().StringVarP(&region, "region", "r", "ap-southeast-1", "Region name")

	parent.AddCommand(cmd)
}

func run(_ []string) {
	flag.Parse()

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	// Create a cache
	cacheSnapshot := cache.NewSnapshotCache(false, cache.IDHash{}, logrus.StandardLogger())

	// Watch Kubernetes resources for changes endpoints and snapshot
	clientSet := xds.NewK8sClient()
	logrus.Infof("Listening on service %v", services)
	//go WatchResource(clientSet, cacheSnapshot)
	go UpdateResource(clientSet, cacheSnapshot)

	// Run the xDS server
	ctx := context.Background()
	cb := &xds.Callbacks{}
	srv := server.NewServer(ctx, cacheSnapshot, cb)
	go xds.RunServer(ctx, srv, port)

	go httpserver.ServeHTTP()
	signal := make(chan struct{})
	<-signal
	cb.Report()
}

func WatchResource(clientSet *kubernetes.Clientset, cacheSnapshot cache.SnapshotCache) {
	endpoints := make(chan map[string][]xds.PodEndPoint, 100)
	ctx := context.Background()
	go xds.ListenNamespaces(clientSet, services, namespace, endpoints)
	for {
		select {
		case k8sEndpoint := <-endpoints:
			snapshot, err := xds.GenerateSnapshot(k8sEndpoint, region, zone)
			if err != nil {
				logrus.Fatalf("Error generating snapshot %v", err)
			}
			if err = cacheSnapshot.SetSnapshot(ctx, nodeID, snapshot); err != nil {
				logrus.Fatalf("Error set snapshot %v", err)
			}
		}
	}
}

func UpdateResource(clientSet *kubernetes.Clientset, cacheSnapshot cache.SnapshotCache) {
	ctx := context.Background()
	for {
		k8sEndpoint := xds.ListEndpoints(clientSet, services, namespace)
		snapshot, err := xds.GenerateSnapshot(k8sEndpoint, region, zone)
		if err != nil {
			logrus.Fatalf("Error generating snapshot %v", err)
		}
		if err = cacheSnapshot.SetSnapshot(ctx, nodeID, snapshot); err != nil {
			logrus.Fatalf("Error set snapshot %v", err)
		}
		time.Sleep(60 * time.Second)
	}
}
