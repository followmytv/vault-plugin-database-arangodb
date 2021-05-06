package arangodb

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/helper/testhelpers/docker"
	dbplugin "github.com/hashicorp/vault/sdk/database/dbplugin/v5"
	dbtesting "github.com/hashicorp/vault/sdk/database/dbplugin/v5/testing"
)

const arangoDatabaseGrant = `{"database_grants": [{"db": "db1", "access": "ro"}]}`

const (
	rootUsername = "root"
	rootPassword = "root"
)

type Config struct {
	docker.ServiceURL
	Username string
	Password string
}

var _ docker.ServiceConfig = &Config{}

func (c *Config) clientConfig() driver.ClientConfig {
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{c.URL().String()},
	})

	if err != nil {
		log.Fatalf("Unable to create connection: %s", err)
	}

	return driver.ClientConfig{
		Connection:     conn,
		Authentication: driver.BasicAuthentication(c.Username, c.Password),
	}
}

func copyConfig(config map[string]interface{}) map[string]interface{} {
	newConfig := map[string]interface{}{}
	for k, v := range config {
		newConfig[k] = v
	}
	return newConfig
}

func assertCredsExist(t testing.TB, address, username, password string) {
	t.Helper()
	err := testCredsExist(address, username, password)
	if err != nil {
		t.Fatalf("Could not log in as %q", username)
	}
}

func createClient(address, username, password string) driver.Client {
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{address},
	})

	if err != nil {
		log.Fatalf("Unable to create connection: %s", err)
	}

	conf := driver.ClientConfig{
		Connection:     conn,
		Authentication: driver.BasicAuthentication(username, password),
	}

	client, err := driver.NewClient(conf)
	if err != nil {
		log.Fatalf("error creating ArangoDB client: %s", err)
	}

	return client
}

func testCredsExist(address, username, password string) error {
	client := createClient(address, username, password)

	_, err := client.User(context.Background(), username)
	if err != nil {
		return err
	}

	return nil
}

func createDatabase(address, username, password, database string) error {
	client := createClient(address, username, password)
	_, err := client.CreateDatabase(context.Background(), database, &driver.CreateDatabaseOptions{})
	return err
}

func prepareArangoDBTestContainer(t *testing.T) (func(), *Config) {
	c := &Config{
		Username: "root",
		Password: "root",
	}
	if host := os.Getenv("ARANGODB_HOST"); host != "" {
		c.ServiceURL = *docker.NewServiceURL(url.URL{Scheme: "http", Host: host})
		return func() {}, c
	}

	runner, err := docker.NewServiceRunner(docker.RunOptions{
		ImageRepo: "arangodb",
		ImageTag:  "3.7.11",
		Env: []string{
			"ARANGO_ROOT_PASSWORD=" + c.Password,
		},
		Ports: []string{"8529/tcp"},
	})
	if err != nil {
		t.Fatalf("Could not start docker ArangoDB: %s", err)
	}
	svc, err := runner.StartService(context.Background(), func(ctx context.Context, host string, port int) (docker.ServiceConfig, error) {
		c.ServiceURL = *docker.NewServiceURL(url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:%d", host, port),
		})

		client, err := driver.NewClient(c.clientConfig())
		if err != nil {
			return nil, errwrap.Wrapf("error creating ArangoDB client: {{err}}", err)
		}

		_, err = client.Version(ctx)
		if err != nil {
			return nil, errwrap.Wrapf("error checking cluster status: {{err}}", err)
		}

		return c, nil
	})
	if err != nil {
		t.Fatalf("Could not start docker ArangoDB: %s", err)
	}

	return svc.Cleanup, svc.Config.(*Config)
}

func TestArangoDB_Initialize(t *testing.T) {
	cleanup, config := prepareArangoDBTestContainer(t)
	defer cleanup()

	db := new()
	defer dbtesting.AssertClose(t, db)

	pluginConfig := map[string]interface{}{
		"connection_url": config.URL().String(),
		"username":       rootUsername,
		"password":       rootPassword,
	}

	// Make a copy since the original map could be modified by the Initialize call
	expectedConfig := copyConfig(pluginConfig)

	req := dbplugin.InitializeRequest{
		Config:           pluginConfig,
		VerifyConnection: true,
	}

	resp := dbtesting.AssertInitialize(t, db, req)

	if !reflect.DeepEqual(resp.Config, expectedConfig) {
		t.Fatalf("Actual config: %#v\nExpected config: %#v", resp.Config, expectedConfig)
	}

	if !db.Initialized {
		t.Fatal("Database should be initialized")
	}
}

// Tests creation of a user without specifying any create statements, it should
// result in a user without any permissions set on databases or collections.
func TestArangoDB_CreateUser_Default(t *testing.T) {
	cleanup, config := prepareArangoDBTestContainer(t)
	defer cleanup()

	db := new()
	defer dbtesting.AssertClose(t, db)

	initReq := dbplugin.InitializeRequest{
		Config: map[string]interface{}{
			"connection_url": config.URL().String(),
			"username":       rootUsername,
			"password":       rootPassword,
		},
		VerifyConnection: true,
	}
	dbtesting.AssertInitialize(t, db, initReq)

	password := "myreallysecurepassword"
	createReq := dbplugin.NewUserRequest{
		UsernameConfig: dbplugin.UsernameMetadata{
			DisplayName: "test",
			RoleName:    "test",
		},
		Statements: dbplugin.Statements{
			Commands: []string{},
		},
		Password:   password,
		Expiration: time.Now().Add(time.Minute),
	}
	createResp := dbtesting.AssertNewUser(t, db, createReq)
	assertCredsExist(t, config.URL().String(), createResp.Username, password)
}

// Tests creation of a user without specifying any create statements, it should
// result in a user without any permissions set on databases or collections.
func TestArangoDB_CreateUser_DatabaseGrant(t *testing.T) {
	cleanup, config := prepareArangoDBTestContainer(t)
	defer cleanup()

	// Make sure we create our neccesary database
	err := createDatabase(config.URL().String(), rootUsername, rootPassword, "db1")
	if err != nil {
		t.Fatalf("Failed to create new database: %s", err)
	}

	db := new()
	defer dbtesting.AssertClose(t, db)

	initReq := dbplugin.InitializeRequest{
		Config: map[string]interface{}{
			"connection_url": config.URL().String(),
			"username":       rootUsername,
			"password":       rootPassword,
		},
		VerifyConnection: true,
	}
	dbtesting.AssertInitialize(t, db, initReq)

	password := "new-passwd"
	createReq := dbplugin.NewUserRequest{
		UsernameConfig: dbplugin.UsernameMetadata{
			DisplayName: "test",
			RoleName:    "test",
		},
		Statements: dbplugin.Statements{
			Commands: []string{arangoDatabaseGrant},
		},
		Password:   password,
		Expiration: time.Now().Add(time.Minute),
	}
	createResp := dbtesting.AssertNewUser(t, db, createReq)
	assertCredsExist(t, config.URL().String(), createResp.Username, password)
}
