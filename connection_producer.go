package arangodb

import (
	"context"
	"sync"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/hashicorp/vault/sdk/database/helper/connutil"
)

// arangoDBConnectionProducer implements ConnectionProducer and provides an
// interface for databases to make connections.
type arangoDBConnectionProducer struct {
	Username      string `json:"username" structs:"username" mapstructure:"username"`
	Password      string `json:"password" structs:"password" mapstructure:"password"`
	ConnectionURL string `json:"connection_url" structs:"connection_url" mapstructure:"connection_url"`

	Initialized bool
	// RawConfig    map[string]interface{}
	Type         string
	clientConfig *driver.ClientConfig
	client       driver.Client
	sync.Mutex
}

// Connection creates a database connection
func (c *arangoDBConnectionProducer) Connection(ctx context.Context) (interface{}, error) {
	if !c.Initialized {
		return nil, connutil.ErrNotInitialized
	}

	client, err := createClient(ctx, "http://localhost:8529", c.clientConfig)
	if err != nil {
		return nil, err
	}
	c.client = client
	return c.client, nil
}

func createClient(ctx context.Context, connURL string, clientConfig *driver.ClientConfig) (client driver.Client, err error) {
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{connURL},
	})
	if err != nil {
		return nil, err
	}

	c, err := driver.NewClient(driver.ClientConfig{
		Connection: conn,
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Close terminates the database connection.
func (c *arangoDBConnectionProducer) Close() error {
	c.Lock()
	defer c.Unlock()

	c.client = nil

	return nil
}

func (c *arangoDBConnectionProducer) secretValues() map[string]string {
	return map[string]string{
		c.Password: "[password]",
	}
}
