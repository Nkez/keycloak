package main

import (
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"keylock_test/proto"
	"keylock_test/repository/keycloak"
	"log"
	"net"
)

func initConfig() {
	var configPath string

	flag.StringVar(&configPath, "cf", ".", "Path to the config file")
	flag.Parse()

	viper.SetConfigFile("config/config.yaml")
	viper.AddConfigPath(configPath)

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("No valid config file is provided: %s", err.Error()))
	}
}

func main() {
	initConfig()
	lis, err := net.Listen("tcp", viper.GetString("grpc"))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterUserServiceServer(
		grpcServer,
		keycloak.NewKcProvision(
			&keycloak.KcProvisionOpts{
				Config: keycloak.Config{
					MasterRealm:   viper.GetString("keycloak.realm"),
					AdminUsername: viper.GetString("keycloak.username"),
					AdminPassword: viper.GetString("keycloak.password"),
					KeycloakURI:   viper.GetString("keycloak.uri"),
				},
			},
		))

	reflection.Register(grpcServer)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %s", err)
	}
}
