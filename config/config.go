package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type PostgresConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (c PostgresConfig) PgxConnectionInfo(maxConn int, maxConnLifeDuration string) string {
	if c.Password == "" {
		return fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable pool_max_conns=%d pool_max_conn_lifetime=%s",
			c.Host,
			"5432",
			c.User,
			c.Name,
			maxConn,
			maxConnLifeDuration,
		)
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable pool_max_conns=%d pool_max_conn_lifetime=%s",
		c.Host,
		"5432",
		c.User,
		c.Password,
		c.Name,
		maxConn,
		maxConnLifeDuration,
	)
}

func New(path string) (*PostgresConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	conf := PostgresConfig{}
	if err := decoder.Decode(&conf); err != nil {
		return nil, err
	}
	return &conf, nil
}
