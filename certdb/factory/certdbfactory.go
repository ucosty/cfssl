package certdbfactory

import (
	// "database/sql"
	"encoding/json"
	"github.com/ucosty/cfssl/certdb"
	"github.com/ucosty/cfssl/certdb/couchbase"
    // "github.com/ucosty/cfssl/certdb/consul"
	// cfsslsql "github.com/ucosty/cfssl/certdb/sql"
	"io/ioutil"
)

func NewAccessor(config string) certdb.Accessor {
	var options map[string]string
	body, err := ioutil.ReadFile(config)
	if err != nil {
		return nil
	}
	json.Unmarshal(body, &options)

	switch options["engine"] {
	case "couchbase":
		return couchbase.NewAccessor(config)
	// case "sql":
	// 	db, err := sql.Open(options["driver"], options["data_source"])
	// 	if err == nil {
	// 		return cfsslsql.NewAccessor(db)
	// 	}
	}
	return nil
}
