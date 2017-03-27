package dbconf

import (
	"encoding/json"
	"errors"
	"io/ioutil"

	cferr "github.com/ucosty/cfssl/errors"
	"github.com/ucosty/cfssl/log"
	"github.com/jmoiron/sqlx"
)

// DBConfig contains the database driver name and configuration to be passed to Open
type DBConfig struct {
	DriverName     string `json:"driver"`
	DataSourceName string `json:"data_source"`
}

// LoadFile attempts to load the db configuration file stored at the path
// and returns the configuration. On error, it returns nil.
func LoadFile(path string) (cfg *DBConfig, err error) {
	log.Debugf("loading db configuration file from %s", path)
	if path == "" {
		return nil, cferr.Wrap(cferr.PolicyError, cferr.InvalidPolicy, errors.New("invalid path"))
	}

	var body []byte
	body, err = ioutil.ReadFile(path)
	if err != nil {
		return nil, cferr.Wrap(cferr.PolicyError, cferr.InvalidPolicy, errors.New("could not read configuration file"))
	}

	cfg = &DBConfig{}
	err = json.Unmarshal(body, &cfg)
	if err != nil {
		return nil, cferr.Wrap(cferr.PolicyError, cferr.InvalidPolicy,
			errors.New("Failed to unmarshal configuration: "+err.Error()))
	}

	if cfg.DataSourceName == "" || cfg.DriverName == "" {
		return nil, cferr.Wrap(cferr.PolicyError, cferr.InvalidPolicy, errors.New("Invalid DB configuration"))
	}

	return
}

func DBFromConfig(path string) (db *sqlx.DB, err error) {
	return nil, nil
}
