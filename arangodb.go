package arangodb

import (
	"context"
	"errors"
	"fmt"

	driver "github.com/arangodb/go-driver"
	dbplugin "github.com/hashicorp/vault/sdk/database/dbplugin/v5"
	"github.com/hashicorp/vault/sdk/helper/strutil"
	"github.com/hashicorp/vault/sdk/helper/template"
)

const (
	arangoDBTypeName        = "arangodb"
	defaultUsernameTemplate = `{{ printf "v-%s-%s-%s-%s" (.DisplayName | truncate 15) (.RoleName | truncate 15) (random 20) (unix_time) | truncate 100 }}`
)

// ArangoDB is an implementation of Database interface
type ArangoDB struct {
	*arangoDBConnectionProducer

	usernameProducer template.StringTemplate
}

var _ dbplugin.Database = &ArangoDB{}

// New configures and returns Mock backends
func New() (interface{}, error) {
	db := new()
	dbType := dbplugin.NewDatabaseErrorSanitizerMiddleware(db, db.secretValues)
	return dbType, nil
}

func new() *ArangoDB {
	connProducer := &arangoDBConnectionProducer{
		Type: arangoDBTypeName,
	}

	return &ArangoDB{
		arangoDBConnectionProducer: connProducer,
	}
}

// Type returns the TypeName for this backend
func (a *ArangoDB) Type() (string, error) {
	return arangoDBTypeName, nil
}

// Initialize initializes the db plugin
func (a *ArangoDB) Initialize(ctx context.Context, req dbplugin.InitializeRequest) (dbplugin.InitializeResponse, error) {
	a.Lock()
	defer a.Unlock()

	usernameTemplate, err := strutil.GetString(req.Config, "username_template")
	if err != nil {
		return dbplugin.InitializeResponse{}, fmt.Errorf("failed to retrieve username_template: %w", err)
	}
	if usernameTemplate == "" {
		usernameTemplate = defaultUsernameTemplate
	}

	up, err := template.NewTemplate(template.Template(usernameTemplate))
	if err != nil {
		return dbplugin.InitializeResponse{}, fmt.Errorf("unable to initialize username template: %w", err)
	}
	a.usernameProducer = up

	_, err = a.usernameProducer.Generate(dbplugin.UsernameMetadata{})
	if err != nil {
		return dbplugin.InitializeResponse{}, fmt.Errorf("invalid username template: %w", err)
	}

	// Set initialized to true at this point since all fields are set,
	// and the connection can be established at a later time.
	a.Initialized = true

	if req.VerifyConnection {
		_, err := a.Connection(ctx)
		if err != nil {
			return dbplugin.InitializeResponse{}, fmt.Errorf("failed to verify connection: %w", err)
		}

		_, err = a.client.Version(ctx)
		if err != nil {
			return dbplugin.InitializeResponse{}, fmt.Errorf("failed to verify connection: %w", err)
		}
	}

	resp := dbplugin.InitializeResponse{
		Config: req.Config,
	}
	return resp, nil
}

// DeleteUser deletes a user account
func (a *ArangoDB) DeleteUser(ctx context.Context, req dbplugin.DeleteUserRequest) (dbplugin.DeleteUserResponse, error) {
	a.Lock()
	defer a.Unlock()

	user, err := a.client.User(ctx, req.Username)
	if err != nil {
		return dbplugin.DeleteUserResponse{}, err
	}

	err = user.Remove(ctx)
	return dbplugin.DeleteUserResponse{}, err
}

// NewUser creates a new user account
func (a *ArangoDB) NewUser(ctx context.Context, req dbplugin.NewUserRequest) (dbplugin.NewUserResponse, error) {
	a.Lock()
	defer a.Unlock()

	username, err := a.usernameProducer.Generate(req.UsernameConfig)
	if err != nil {
		return dbplugin.NewUserResponse{}, err
	}

	options := driver.UserOptions{
		Password: req.Password,
	}
	user, err := a.client.CreateUser(ctx, username, &options)
	if err != nil {
		return dbplugin.NewUserResponse{}, err
	}

	resp := dbplugin.NewUserResponse{
		Username: user.Name(),
	}

	return resp, nil
}

// UpdateUser updates a user account
func (a *ArangoDB) UpdateUser(ctx context.Context, req dbplugin.UpdateUserRequest) (dbplugin.UpdateUserResponse, error) {
	a.Lock()
	defer a.Unlock()

	return dbplugin.UpdateUserResponse{}, errors.New("Not implemented")
}
