package libsql

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tursodatabase/go-libsql"
	"gorm.io/gorm"
)

var tempDir string

func dbUrl() string {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "http://localhost:18080"
	}
	return url
}

func TestGormOpen(t *testing.T) {
	setup(t)
	defer teardown(t)

	dbPath := filepath.Join(tempDir, "embedded_replica.db")
	connector, err := libsql.NewEmbeddedReplicaConnector(dbPath, dbUrl())
	if err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		name   string
		config Config
	}{
		{"local", Config{DSN: fmt.Sprintf("file:%s", filepath.Join(tempDir, "open_local.db"))}},
		{"remote", Config{DSN: dbUrl()}},
		{"embedded replica", Config{Conn: sql.OpenDB(connector)}},
	}

	for _, tc := range testCases {
		if tc.config.DSN != "" {
			t.Run(tc.name+" (open)", func(t *testing.T) {
				_, err := gorm.Open(Open(tc.config.DSN), &gorm.Config{})
				if err != nil {
					t.Fatal(err)
				}
			})
		}
		t.Run(tc.name+" (new w/dsn)", func(t *testing.T) {
			_, err := gorm.Open(New(tc.config), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
		})
	}

	t.Run("local (new w/conn)", func(t *testing.T) {
		dbPath := filepath.Join(tempDir, "open_local_ext.db")
		db, err := sql.Open("libsql",
			fmt.Sprintf("file:%s", dbPath))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		_, err = gorm.Open(New(Config{Conn: db}), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("remote (new w/conn)", func(t *testing.T) {
		db, err := sql.Open("libsql", dbUrl())
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		_, err = gorm.Open(New(Config{Conn: db}), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("embedded replica (new w/conn)", func(t *testing.T) {
		dbPath := filepath.Join(tempDir, "open_embedded_ext.db")
		conn, err := libsql.NewEmbeddedReplicaConnector(dbPath, dbUrl())
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		db := sql.OpenDB(conn)
		defer db.Close()
		_, err = gorm.Open(New(Config{Conn: db}), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
	})
}

// Simple query as sanity check, however it is assumed
// libsql has the same semantics as sqlite3.
// If you hit an error that has to do with libsql not
// matching sqlite, please open an issue with turso's go-libsql.
func TestSanity(t *testing.T) {
	setup(t)
	defer teardown(t)

	conns := []struct {
		name string
		db   *gorm.DB
	}{
		{"local", func() *gorm.DB {
			g, err := gorm.Open(Open("file:"+filepath.Join(tempDir, "sanity_local.db")), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			return g
		}()},
		{"remote", func() *gorm.DB {
			g, err := gorm.Open(Open(dbUrl()), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			return g
		}()},
		{"embedded replica", func() *gorm.DB {
			dbPath := filepath.Join(tempDir, "embedded_replica.db")
			connector, err := libsql.NewEmbeddedReplicaConnector(dbPath, dbUrl())
			if err != nil {
				t.Fatal(err)
			}

			g, err := gorm.Open(New(Config{DSN: dbUrl(), Conn: sql.OpenDB(connector)}), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			return g
		}()},
	}

	for _, conn := range conns {
		t.Run(conn.name, func(t *testing.T) {
			db := conn.db
			t.Run("select 1", func(t *testing.T) {
				var num int
				tx := db.Raw("SELECT 1").Scan(&num)
				if tx.Error != nil {
					t.Fatal(tx.Error)
				}
				if num != 1 {
					t.Fatal("expected 1, got", num)
				}
			})
			tableName := fmt.Sprintf("foo_%d", time.Now().UnixNano())
			t.Run("create table", func(t *testing.T) {
				tx := db.Exec(fmt.Sprintf(
					"CREATE TABLE %s (id INTEGER PRIMARY KEY, name TEXT)", tableName))
				if tx.Error != nil {
					t.Fatal(tx.Error)
				}
			})
			type model struct {
				ID   int `gorm:"primaryKey"`
				Name string
			}
			t.Run("insert", func(t *testing.T) {
				m1 := model{ID: int(time.Now().UnixMilli()), Name: "m1"}
				res := db.Table(tableName).Create(&m1)
				if res.Error != nil {
					t.Fatal(res.Error)
				}
				var m1_ck = model{}
				res = db.Raw("SELECT * FROM "+tableName+" WHERE id = ?", m1.ID).Scan(&m1_ck)
				if res.Error != nil {
					t.Fatal(res.Error)
				}
				if m1_ck.ID != m1.ID {
					t.Fatal("expected", m1.ID, "got", m1_ck.ID)
				}
				if m1_ck.Name != m1.Name {
					t.Fatal("expected", m1.Name, "got", m1_ck.Name)
				}
			})
			t.Run("select where", func(t *testing.T) {
				var m2 = model{ID: int(time.Now().UnixMilli()), Name: "m2"}
				res := db.Exec("INSERT INTO "+tableName+" (id, name) VALUES (?, ?)", m2.ID, m2.Name)
				if res.Error != nil {
					t.Fatal(res.Error)
				}
				var m2_ck model
				res = db.Table(tableName).Where("id = ?", m2.ID).Find(&m2_ck)
				if res.Error != nil {
					t.Fatal(res.Error)
				}
				if m2_ck.ID != m2.ID {
					t.Fatal("expected", m2.ID, "got", m2_ck.ID)
				}
				if m2_ck.Name != m2.Name {
					t.Fatal("expected", m2.Name, "got", m2_ck.Name)
				}
			})
			db.Exec("DROP TABLE " + tableName)
		})
	}
}

func TestEmbeddedSync(t *testing.T) {
	setup(t)
	defer teardown(t)

	dbPath := filepath.Join(tempDir, "embedded_replica.db")
	connector, err := libsql.NewEmbeddedReplicaConnector(dbPath, dbUrl())
	if err != nil {
		t.Fatal(err)
	}

	db, err := gorm.Open(New(Config{DSN: dbUrl(), Conn: sql.OpenDB(connector)}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	db.Raw(("SELECT 1"))
	_, err = connector.Sync()
	if err != nil {
		t.Fatal(err)
	}

}
func setup(t *testing.T) {
	dir, err := os.MkdirTemp("", "libsql-*")
	if err != nil {
		t.Fail()
	}
	tempDir = dir
}

func teardown(t *testing.T) {
	err := os.RemoveAll(tempDir)
	if err != nil {
		t.Fail()
	}
	tempDir = ""
}
