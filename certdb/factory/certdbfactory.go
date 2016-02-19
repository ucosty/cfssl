package certdbfactory

import (
	"io/ioutil"
	"encoding/json"
	"database/sql"
	"github.com/cloudflare/cfssl/certdb"
	"github.com/cloudflare/cfssl/certdb/couchbase"
	cfsslsql "github.com/cloudflare/cfssl/certdb/sql"
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
    case "sql":
    	db, err := sql.Open(options["driver"], options["data_source"])
    	if err == nil {
    		return cfsslsql.NewAccessor(db)
    	}
    }
    return nil
}
