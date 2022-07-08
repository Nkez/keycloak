package keycloak

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/Nerzal/gocloak/v11"
	date_protobuf "github.com/Nkez/date-protobuf"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"google.golang.org/protobuf/types/known/emptypb"
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

func (c *Keycloak) Create(ctx context.Context, create *date_protobuf.CreateUser) (*emptypb.Empty, error) {
	ctx = context.Background()
	keyCloakConfig := c.Config
	keyCloakClient := keyCloakConfig.newKeycloakClient()
	keyCloakToken, err := keyCloakConfig.newKeycloakToken(ctx, keyCloakClient)
	if err != nil {
		return nil, err
	}
	user := gocloak.User{
		Username:  gocloak.StringP(create.UserName),
		Enabled:   gocloak.BoolP(create.Enabled),
		FirstName: gocloak.StringP(create.FirstName),
		LastName:  gocloak.StringP(create.LastName),
		Email:     gocloak.StringP(create.Email),
		Attributes: &map[string][]string{
			"phone": gocloak.StringOrArray{
				create.MobilePhone,
			},
			"country": gocloak.StringOrArray{
				create.Country,
			},
			"photo": gocloak.StringOrArray{
				create.Photo,
			},
		},
	}
	u, err := keyCloakClient.CreateUser(ctx, keyCloakToken.AccessToken, c.MasterRealm, user)
	if err != nil {
		return nil, err
	}
	fmt.Println(u)
	return &emptypb.Empty{}, nil
}

func (c *Keycloak) Update(ctx context.Context, update *date_protobuf.User) (*emptypb.Empty, error) {
	ctx = context.Background()
	keyCloakConfig := c.Config
	keyCloakClient := keyCloakConfig.newKeycloakClient()
	keyCloakToken, err := keyCloakConfig.newKeycloakToken(ctx, keyCloakClient)
	if err != nil {
		return nil, err
	}
	user := decodeUpdateUser(update)
	err = keyCloakClient.UpdateUser(ctx, keyCloakToken.AccessToken, c.MasterRealm, user)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func decodeUpdateUser(update *date_protobuf.User) gocloak.User {
	var user gocloak.User
	var country string
	var photo string
	var phone string
	user.ID = gocloak.StringP(update.Id)
	if update.Email != "" {
		user.Email = gocloak.StringP(update.Email)
	}
	if update.FirstName != "" {
		user.FirstName = gocloak.StringP(update.FirstName)
	}
	if update.LastName != "" {
		user.LastName = gocloak.StringP(update.LastName)
	}
	if update.UserName != "" {
		user.Username = gocloak.StringP(update.UserName)
	}

	user.Attributes = &map[string][]string{
		"phone": gocloak.StringOrArray{
			phone,
		},
		"country": gocloak.StringOrArray{
			country,
		},
		"photo": gocloak.StringOrArray{
			photo,
		},
	}
	return user
}

func (c *Keycloak) Get(ctx context.Context, id *date_protobuf.GetUser) (*date_protobuf.User, error) {
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

func (c *Keycloak) Delete(ctx context.Context, id *date_protobuf.GetUser) (*emptypb.Empty, error) {
	keyCloakConfig := c.Config
	keyCloakClient := keyCloakConfig.newKeycloakClient()
	keyCloakToken, err := keyCloakConfig.newKeycloakToken(ctx, keyCloakClient)
	if err != nil {
		return nil, err
	}
	err = keyCloakClient.DeleteUser(ctx, keyCloakToken.AccessToken, c.MasterRealm, id.Id)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

type User struct {
	ID        string  `db:"id" json:"id"`
	FirstName string  `db:"first_name" json:"fn"`
	LastName  string  `db:"last_name" json:"last_name"`
	Email     string  `db:"email" json:"email"`
	Username  string  `db:"username" json:"username"`
	Info      []uint8 `db:"info" json:"info"`
	Enabled   bool    `db:"enabled" json:"enabled"`
}

type Info struct {
	Country string `db:"country" json:"country"`
	Phone   string `db:"phone" json:"phone"`
	Photo   string `db:"photo" json:"photo"`
}

func (c *Keycloak) List(ctx context.Context, filter *date_protobuf.Filter) (*date_protobuf.ListUser, error) {
	users := make([]User, 0, defaultPageSize)
	query := `	select  ue.id ,ue.first_name  ,ue.last_name , ue.email , ue.username, 
	        	json_object_agg(ua."name" , ua.value) FILTER (WHERE ua."name"  IS NOT NULL) as info,
	        	ue.enabled
				FROM keycloak_role kr
				JOIN user_role_mapping rm ON kr.id = rm.role_id
				JOIN user_entity ue ON rm.user_id = ue.id
				left join user_attribute ua on ue.id  = ua.user_id`
	query, args := decodeFilter(query, filter, c.DB)
	query = fmt.Sprintf("%s and (ua.name = 'country' or ua.name = 'phone' or ua.name = 'photo') group by ue.id", query)
	query = paginateFilter(query, filter)
	if err := c.DB.Select(&users, query, args...); err != nil {
		return nil, err
	}
	response := &date_protobuf.ListUser{
		Users: make([]*date_protobuf.User, 0, len(users)),
	}
	for _, e := range users {
		decode, err := decodeToGRPC(&e)
		if err != nil {
			return nil, err
		}
		response.Users = append(response.Users, decode)
	}
	return response, nil
}

func decodeToGRPC(user *User) (*date_protobuf.User, error) {
	var info *Info
	err := json.Unmarshal(user.Info, &info)
	if err != nil {
		return nil, err
	}
	return &date_protobuf.User{
		Id:          user.ID,
		LastName:    user.LastName,
		FirstName:   user.FirstName,
		UserName:    user.Username,
		Email:       user.Email,
		MobilePhone: info.Phone,
		Country:     info.Country,
		Photo:       info.Photo,
		Enabled:     user.Enabled,
	}, nil
}

func decodeFilter(query string, filter *date_protobuf.Filter, db *sqlx.DB) (string, []interface{}) {
	query = fmt.Sprintf("%s WHERE 1=1", query)
	args := make([]interface{}, 0)
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
	if filter.Enabled == false {
		query = fmt.Sprintf("%s AND ue.enabled = (?)", query)
		args = append(args, &filter.Enabled)
	}
	if filter.Enabled == true {
		query = fmt.Sprintf("%s AND ue.enabled = (?)", query)
		args = append(args, &filter.Enabled)
	}
	query = db.Rebind(query)
	return query, args
}

func paginateFilter(query string, filter *date_protobuf.Filter) string {
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

func keyCloakToGRPC(user *gocloak.User) *date_protobuf.User {
	var p string
	var c string
	var photo string
	if user.Attributes != nil {
		phone, ok := (*user.Attributes)["phone"]
		if ok {
			p = strings.Join(phone[:], ",")
		}
		country, ok := (*user.Attributes)["country"]
		if ok {
			c = strings.Join(country[:], ",")
		}
		photo, ok := (*user.Attributes)["photo"]
		if ok {
			c = strings.Join(photo[:], ",")
		}
	} else {
		p = ""
		c = ""
		photo = ""
	}
	return &date_protobuf.User{
		Id:          PString(user.ID),
		LastName:    PString(user.LastName),
		FirstName:   PString(user.FirstName),
		UserName:    PString(user.Username),
		Email:       PString(user.Email),
		MobilePhone: p,
		Country:     c,
		Photo:       photo,
		Enabled:     *user.Enabled,
	}
}
