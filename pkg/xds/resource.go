package xds

import (
	"fmt"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	routerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"net"
	"strconv"
	"sync/atomic"
)

var Version int64

func clusterLoadAssignment(podEndPoints []PodEndPoint, fullName, portName string, port int32) []types.Resource {
	var (
		lbs            []*endpoint.LbEndpoint
		targetHostPort string
	)

	for _, podEndPoint := range podEndPoints {
		port = podEndPoint.Port
		portName = podEndPoint.PortName
		hst := &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address:  podEndPoint.IP,
					Protocol: core.SocketAddress_TCP,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(podEndPoint.Port),
					},
				},
			},
		}

		lbs = append(lbs, &endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address:  hst,
					Hostname: fullName,
				}},
			HealthStatus: core.HealthStatus_HEALTHY,
		})
	}

	if portName == "" {
		targetHostPort = fmt.Sprintf("%s:%d", fullName, port)
	} else {
		targetHostPort = fmt.Sprintf("%s:%s", fullName, portName)
	}
	eds := []types.Resource{
		&endpoint.ClusterLoadAssignment{
			ClusterName: targetHostPort,
			Endpoints: []*endpoint.LocalityLbEndpoints{{
				LoadBalancingWeight: wrapperspb.UInt32(1),
				Locality:            &core.Locality{},
				LbEndpoints:         lbs,
			}},
		},
	}
	return eds
}

func createCluster(targetHostPort string) []types.Resource {
	cls := []types.Resource{
		&cluster.Cluster{
			Name:                 targetHostPort,
			ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
			LbPolicy:             cluster.Cluster_ROUND_ROBIN,
			EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
				EdsConfig: &core.ConfigSource{
					ConfigSourceSpecifier: &core.ConfigSource_Ads{
						Ads: &core.AggregatedConfigSource{},
					},
				},
			},
		},
	}
	return cls
}

func createRouteConfig(fullName, targetHostPort, targetHostPortNumber, endpointName string) *route.RouteConfiguration {
	return &route.RouteConfiguration{
		Name: targetHostPortNumber,
		VirtualHosts: []*route.VirtualHost{
			{
				Name:    targetHostPort,
				Domains: []string{fullName, targetHostPort, targetHostPortNumber, endpointName},
				Routes: []*route.Route{{
					Name: "default",
					Match: &route.RouteMatch{
						PathSpecifier: &route.RouteMatch_Prefix{},
					},
					Action: &route.Route_Route{
						Route: &route.RouteAction{
							ClusterSpecifier: &route.RouteAction_Cluster{
								Cluster: targetHostPort,
							},
						},
					},
				}},
			},
		},
	}
}

func createRoute(route *route.RouteConfiguration) []types.Resource {
	return []types.Resource{
		route,
	}
}

func createListener(targetHostPortNumber string, routeConfig *route.RouteConfiguration) []types.Resource {
	router, _ := anypb.New(&routerv3.Router{})
	manager := &hcm.HttpConnectionManager{
		HttpFilters: []*hcm.HttpFilter{
			{
				Name: wellknown.Router,
				ConfigType: &hcm.HttpFilter_TypedConfig{
					TypedConfig: router,
				},
			},
		},
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: routeConfig,
		},
	}

	pbst, err := anypb.New(manager)
	if err != nil {
		panic(err)
	}

	lds := []types.Resource{
		&listener.Listener{
			Name: targetHostPortNumber,
			ApiListener: &listener.ApiListener{
				ApiListener: pbst,
			},
		},
	}
	return lds
}

func GenerateSnapshot(k8sEndPoints map[string][]PodEndPoint, region, zone string) (*cache.Snapshot, error) {
	var eds, cds, rds, lds []types.Resource
	for _, podEndPoints := range k8sEndPoints {
		if len(podEndPoints) == 0 {
			continue
		}
		ep := podEndPoints[0]
		fullName := fmt.Sprintf("%s.%s", ep.Name, ep.Namespace)
		targetHostPort := net.JoinHostPort(fullName, ep.PortName)
		targetHostPortNumber := net.JoinHostPort(fullName, strconv.Itoa(int(ep.Port)))

		eds = append(eds, clusterLoadAssignment(podEndPoints, fullName, ep.PortName, ep.Port)...)
		routeCfg := createRouteConfig(fullName, targetHostPort, targetHostPortNumber, ep.Name)
		cds = append(cds, createCluster(targetHostPort)...)
		rds = append(rds, createRoute(routeCfg)...)
		lds = append(lds, createListener(targetHostPortNumber, routeCfg)...)
	}
	atomic.AddInt64(&Version, 1)

	logrus.Debug("Creating Snapshot", zap.Int64("version", Version), zap.Any("CDS", cds), zap.Any("RDS", rds), zap.Any("LDS", lds))
	resourceMap := map[resource.Type][]types.Resource{
		resource.EndpointType: eds,
		resource.ClusterType:  cds,
		resource.RouteType:    rds,
		resource.ListenerType: lds,
	}
	snapshot, err := cache.NewSnapshot(fmt.Sprint(Version), resourceMap)
	if err != nil {
		return snapshot, err
	}
	//if err := snapshot.Consistent(); err != nil {
	//	logrus.Error("Snapshot inconsistency", zap.Any("snapshot", snapshot), zap.Error(err))
	//	return snapshot, err
	//}
	return snapshot, nil
}
