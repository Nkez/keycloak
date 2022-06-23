package keycloak

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/Nerzal/gocloak/v11"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"google.golang.org/protobuf/types/known/emptypb"
	"keylock_test/proto"
	"strings"
)

const (
	defaultPageSize = 10
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

type User struct {
	ID        string `db:"id" json:"id"`
	FirstName string `db:"first_name" json:"fn"`
	LastName  string `db:"last_name" json:"last_name"`
	Email     string `db:"email" json:"email"`
	Username  string `db:"username" json:"username"`
	Value     string `db:"string_agg" json:"vl"`
}

type UserInfo struct {
	Phone   string
	Country string
}

func (c *Keycloak) List(ctx context.Context, filter *proto.Filter) (*proto.ListUser, error) {
	users := make([]User, 0, defaultPageSize)
	query := `	select  ue.id ,ue.first_name  ,ue.last_name , ue.email , ue.username, string_agg(ua.value, ',')
				FROM keycloak_role kr
				JOIN user_role_mapping rm ON kr.id = rm.role_id
				JOIN user_entity ue ON rm.user_id = ue.id
				join user_attribute ua on ue.id  = ua.user_id`
	query, args := decodeFilter(query, filter, c.DB)
	query = fmt.Sprintf("%s and (ua.name = 'country' or ua.name = 'phone') group by ue.id", query)
	query = paginateFilter(query, filter)
	if err := c.DB.Select(&users, query, args...); err != nil {
		return nil, err
	}
	response := &proto.ListUser{
		Users: make([]*proto.User, 0, len(users)),
	}
	for _, e := range users {
		response.Users = append(response.Users, decodeToGRPC(&e))
	}
	return response, nil
}

func decodeToGRPC(user *User) *proto.User {
	var p string
	var c string
	if user.Value != "," {
		sl := strings.Split(user.Value, ",")
		c = sl[0]
		p = sl[1]
	} else {
		p = ""
		c = ""
	}
	return &proto.User{
		Id:          user.ID,
		LastName:    user.LastName,
		FirstName:   user.FirstName,
		UserName:    user.Username,
		Email:       user.Email,
		MobilePhone: p,
		Country:     c,
	}
}

func decodeFilter(query string, filter *proto.Filter, db *sqlx.DB) (string, []interface{}) {
	query = fmt.Sprintf("%s WHERE 1=1", query)
	args := make([]interface{}, 0)
	fmt.Println(filter.Role)
	if filter.Role != "" {
		query = fmt.Sprintf("%s AND kr.name = (?)", query)
		args = append(args, filter.Role)
	}
	if filter.FirstName != "" {
		query = fmt.Sprintf("%s AND ue.first_name = (?)", query)
		args = append(args, &filter.FirstName)
	}
	if filter.SecondName != "" {
		query = fmt.Sprintf("%s AND ue.last_name = (?)", query)
		args = append(args, &filter.SecondName)
	}
	if filter.Email != "" {
		query = fmt.Sprintf("%s AND ue.email = (?)", query)
		args = append(args, &filter.Email)
	}
	query = db.Rebind(query)
	return query, args
}

func paginateFilter(query string, filter *proto.Filter) string {
	size := int64(defaultPageSize)
	number := int64(1)
	if filter.Size == 0 {
		filter.Size = size
	}
	if filter.Page == 0 {
		filter.Page = number
	}
	if filter.Page > int64(1) {
		query = fmt.Sprintf("%s OFFSET %d", query, (filter.Page-1)*filter.Size)
	}
	return fmt.Sprintf("%s LIMIT %d", query, filter.Size)
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
