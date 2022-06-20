package keycloak

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/Nerzal/gocloak/v11"
	"google.golang.org/protobuf/types/known/emptypb"
	"keylock_test/proto"
	"strings"
)

func (c *Config) newKeycloakClient() gocloak.GoCloak {
	client := gocloak.NewClient(c.KeycloakURI)
	restyClient := client.RestyClient()
	restyClient.SetDebug(true)
	restyClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	return client
}

func (c *Config) newKeycloakToken(ctx context.Context, client gocloak.GoCloak) (*gocloak.JWT, error) {
	return client.LoginAdmin(ctx, c.AdminUsername, c.AdminPassword, c.MasterRealm)
}

func (c *Keycloak) Create(ctx context.Context, create *proto.CreateUser) (*emptypb.Empty, error) {
	keyCloakConfig := c.Config
	keyCloakClient := keyCloakConfig.newKeycloakClient()
	keyCloakToken, err := keyCloakConfig.newKeycloakToken(ctx, keyCloakClient)
	if err != nil {
		return nil, err
	}
	var (
		user = gocloak.User{
			FirstName: gocloak.StringP(create.FirstName),
			LastName:  gocloak.StringP(create.LastName),
			Email:     gocloak.StringP(create.Email),
			Username:  gocloak.StringP(create.UserName),
			Attributes: &map[string][]string{
				"phone": gocloak.StringOrArray{
					create.MobilePhone,
				},
				"country": gocloak.StringOrArray{
					create.Country,
				},
			},
		}
	)

	u, err := keyCloakClient.CreateUser(ctx, keyCloakToken.AccessToken, c.MasterRealm, user)
	if err != nil {
		return nil, err
	}
	fmt.Println(u)
	return &emptypb.Empty{}, nil
}

func (c *Keycloak) Get(ctx context.Context, id *proto.GetUser) (*proto.User, error) {
	keyCloakConfig := c.Config
	keyCloakClient := keyCloakConfig.newKeycloakClient()
	keyCloakToken, err := keyCloakConfig.newKeycloakToken(ctx, keyCloakClient)
	if err != nil {
		return nil, err
	}
	user, err := keyCloakClient.GetUserByID(ctx, keyCloakToken.AccessToken, c.MasterRealm, id.Id)
	if err != nil {
		return nil, err
	}

	return keyCloakToGRPC(user), nil
}
func (c *Keycloak) List(ctx context.Context, empty *emptypb.Empty) (*proto.ListUser, error) {
	keyCloakConfig := c.Config
	keyCloakClient := keyCloakConfig.newKeycloakClient()
	keyCloakToken, err := keyCloakConfig.newKeycloakToken(ctx, keyCloakClient)
	if err != nil {
		return nil, err
	}
	users, err := keyCloakClient.GetUsers(ctx, keyCloakToken.AccessToken, c.MasterRealm, gocloak.GetUsersParams{})
	if err != nil {
		return nil, err
	}
	fmt.Println(users)
	response := &proto.ListUser{
		Users: make([]*proto.User, 0, len(users)),
	}
	for _, e := range users {
		response.Users = append(response.Users, keyCloakToGRPC(e))
	}
	return response, nil
}

func PString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func keyCloakToGRPC(user *gocloak.User) *proto.User {
	var p string
	var c string
	if user.Attributes != nil {
		phone, ok := (*user.Attributes)["phone"]
		if ok {
			p = strings.Join(phone[:], ",")
		}
		country, ok := (*user.Attributes)["country"]
		if ok {
			c = strings.Join(country[:], ",")
		}
	} else {
		p = ""
		c = ""
	}
	return &proto.User{
		Id:          PString(user.ID),
		LastName:    PString(user.LastName),
		FirstName:   PString(user.FirstName),
		UserName:    PString(user.Username),
		Email:       PString(user.Email),
		MobilePhone: p,
		Country:     c,
	}
}
