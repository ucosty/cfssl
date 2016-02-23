// +build postgresql

package sql

import (
	"testing"

	"github.com/ucosty/cfssl/certdb/testdb"
)

func TestPostgreSQL(t *testing.T) {
	db := testdb.PostgreSQLDB()
	dba := NewAccessor(db)
	testEverything(dba, t)
}
