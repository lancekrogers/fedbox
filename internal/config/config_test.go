package config

import (
	"fmt"
	"github.com/go-ap/fedbox/internal/env"
	"os"
	"testing"
)

const (
	hostname = "testing.git"
	logLvl   = "panic"
	secure   = true
	listen   = "127.0.0.3:666"
	pgSQL    = "postgres"
	boltDB   = "boltdb"
	dbHost   = "127.0.0.6"
	dbPort   = 54321
	dbName   = "test"
	dbUser   = "test"
	dbPw     = "pw123+-098"
)

func TestLoadFromEnv(t *testing.T) {
	{
		os.Setenv(KeyDBHost, dbHost)
		os.Setenv(KeyDBPort, fmt.Sprintf("%d", dbPort))
		os.Setenv(KeyDBName, dbName)
		os.Setenv(KeyDBUser, dbUser)
		os.Setenv(KeyDBPw, dbPw)

		os.Setenv(KeyHostname, hostname)
		os.Setenv(KeyLogLevel, logLvl)
		os.Setenv(KeyHTTPS, fmt.Sprintf("%t", secure))
		os.Setenv(KeyListen, listen)
		os.Setenv(KeyStorage, pgSQL)

		var baseURL = fmt.Sprintf("https://%s", hostname)
		c, err := LoadFromEnv(string(env.TEST))
		if err != nil {
			t.Errorf("Error loading env: %s", err)
		}
		db := c.DB
		if db.Host != dbHost {
			t.Errorf("Invalid loaded value for %s: %s, expected %s", KeyDBHost, db.Host, dbHost)
		}
		if db.Port != dbPort {
			t.Errorf("Invalid loaded value for %s: %d, expected %d", KeyDBPort, db.Port, dbPort)
		}
		if db.Name != dbName {
			t.Errorf("Invalid loaded value for %s: %s, expected %s", KeyDBName, db.Name, dbName)
		}
		if db.User != dbUser {
			t.Errorf("Invalid loaded value for %s: %s, expected %s", KeyDBUser, db.User, dbUser)
		}
		if db.Pw != dbPw {
			t.Errorf("Invalid loaded value for %s: %s, expected %s", KeyDBPw, db.Pw, dbPw)
		}

		if c.Host != hostname {
			t.Errorf("Invalid loaded value for %s: %s, expected %s", KeyHostname, c.Host, hostname)
		}
		if c.Secure != secure {
			t.Errorf("Invalid loaded value for %s: %t, expected %t", KeyHTTPS, c.Secure, secure)
		}
		if c.Listen != listen {
			t.Errorf("Invalid loaded value for %s: %s, expected %s", KeyListen, c.Listen, listen)
		}
		if c.Storage != pgSQL {
			t.Errorf("Invalid loaded value for %s: %s, expected %s", KeyStorage, c.Storage, pgSQL)
		}
		if c.BaseURL != baseURL {
			t.Errorf("Invalid loaded BaseURL value: %s, expected %s", c.BaseURL, baseURL)
		}
	}
	{
		os.Setenv(KeyStorage, boltDB)
		c, err := LoadFromEnv(string(env.TEST))
		if err != nil {
			t.Errorf("Error loading env: %s", err)
		}
		var path = fmt.Sprintf("/tmp/%s-%s.bdb", hostname, env.TEST)
		if c.BoltDBPath != path {
			t.Errorf("Invalid loaded BaseURL value: %s, expected %s", c.BoltDBPath, path)
		}
	}
}
