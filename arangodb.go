package arangodb

import (
	"context"
	"errors"

	dbplugin "github.com/hashicorp/vault/sdk/database/dbplugin/v5"
)

const (
	arangoDBTypeName = "arangodb"
)

// ArangoDB is an implementation of Database interface
type ArangoDB struct {
	*arangoDBConnectionProducer
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
func (m *ArangoDB) Type() (string, error) {
	return arangoDBTypeName, nil
}

// Initialize initializes the db plugin
func (m *ArangoDB) Initialize(ctx context.Context, req dbplugin.InitializeRequest) (dbplugin.InitializeResponse, error) {
	m.Lock()
	defer m.Unlock()

	return dbplugin.InitializeResponse{}, errors.New("Not implemented")
}

// DeleteUser deletes a user account
func (m *ArangoDB) DeleteUser(ctx context.Context, req dbplugin.DeleteUserRequest) (dbplugin.DeleteUserResponse, error) {
	m.Lock()
	defer m.Unlock()

	return dbplugin.DeleteUserResponse{}, errors.New("Not implemented")
}

// NewUser creates a new user account
func (m *ArangoDB) NewUser(ctx context.Context, req dbplugin.NewUserRequest) (dbplugin.NewUserResponse, error) {
	m.Lock()
	defer m.Unlock()

	return dbplugin.NewUserResponse{}, errors.New("Not implemented")
}

// UpdateUser updates a user account
func (m *ArangoDB) UpdateUser(ctx context.Context, req dbplugin.UpdateUserRequest) (dbplugin.UpdateUserResponse, error) {
	m.Lock()
	defer m.Unlock()

	return dbplugin.UpdateUserResponse{}, errors.New("Not implemented")
}
