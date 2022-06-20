package keycloak

import "keylock_test/proto"

type Keycloak struct {
	proto.UnimplementedUserServiceServer
	Config
}

type Config struct {
	MasterRealm   string
	AdminUsername string
	AdminPassword string
	KeycloakURI   string
}

type KcProvisionOpts struct {
	Config
}

func NewKcProvision(opt *KcProvisionOpts) *Keycloak {
	return &Keycloak{
		Config: opt.Config,
	}
}
