package keycloak

import (
	"github.com/jmoiron/sqlx"
	"keylock_test/proto"
)

type Keycloak struct {
	proto.UnimplementedUserServiceServer
	Config
}

type Config struct {
	MasterRealm   string
	AdminUsername string
	AdminPassword string
	KeycloakURI   string
	DB            *sqlx.DB
}

type KcProvisionOpts struct {
	Config
}

func NewKcProvision(opt *KcProvisionOpts) *Keycloak {
	return &Keycloak{
		Config: opt.Config,
	}
}
