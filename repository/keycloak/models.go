package keycloak

import (
	date_protobuf "github.com/Nkez/date-protobuf"
	"github.com/jmoiron/sqlx"
)

type Keycloak struct {
	date_protobuf.UnimplementedUserServiceServer
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
