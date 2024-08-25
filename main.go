// main.go
package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	pb "grpcserver/proto" // 这里需要替换为实际的proto文件路径
)

// server is used to implement hello.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello implements hello.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})
	reflection.Register(s)

	// 注册服务到 Nacos
	nacosClient, err := registerToNacos()
	if err != nil {
		log.Fatalf("failed to register service to nacos: %v", err)
	}
	defer nacosClient.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          "127.0.0.1",
		Port:        50051,
		ServiceName: "grpc.hello.service",
		Ephemeral:   true,
	})

	log.Println("gRPC server listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func registerToNacos() (naming_client.INamingClient, error) {
	serverConfigs := []constant.ServerConfig{
		*constant.NewServerConfig("pre.ncs.goat.network", 8848),
	}

	clientConfig := *constant.NewClientConfig(
		constant.WithNamespaceId("public"), // replace with your Nacos namespace
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("./nacos/log"),
		constant.WithCacheDir("./nacos/cache"),
	)

	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)

	if err != nil {
		return nil, err
	}

	// 注册服务到 Nacos
	success, err := namingClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          "127.0.0.1",
		Port:        50051,
		ServiceName: "grpc.hello.service",
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		ClusterName: "DEFAULT", // default cluster name
		GroupName:   "DEFAULT_GROUP", // default group name
	})

	if !success || err != nil {
		return nil, err
	}

	log.Println("Service registered to Nacos successfully")
	return namingClient, nil
}
