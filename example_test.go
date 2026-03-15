package pq_test

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/lib/pq"
)

func Example_open() {
	// Or as URL: postgresql://localhost/pqgo
	db, err := sql.Open("postgres", "dbname=pqgo")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// db.Open() only creates a connection pool, and doesn't actually establish
	// a connection to the database. To ensure the connection works you need to
	// do *something* with a connection.
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}

func Example_openConfig() {
	cfg := pq.Config{
		Host: "localhost",
		Port: 5432,
		User: "pqgo",
	}
	// Or: create a new Config from the defaults, environment, and DSN.
	// cfg, err := pq.NewConfig("host=postgres dbname=pqgo")
	// if err != nil {
	//     log.Fatal(err)
	// }

	c, err := pq.NewConnectorConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Create connection pool.
	db := sql.OpenDB(c)
	defer db.Close()

	// Make sure it works.
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}

func Example_copyFromStdin() {
	// Connect and create table.
	db, err := sql.Open("postgres", "")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`create temp table users (name text, age int)`)
	if err != nil {
		log.Fatal(err)
	}

	// Need to start transaction and prepare a statement.
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare(`copy users (name, age) from stdin`)
	if err != nil {
		log.Fatal(err)
	}

	// Insert rows.
	users := []struct {
		Name string
		Age  int
	}{
		{"Donald Duck", 36},
		{"Scrooge McDuck", 86},
	}
	for _, user := range users {
		_, err = stmt.Exec(user.Name, int64(user.Age))
		if err != nil {
			log.Fatal(err)
		}
	}

	// Finalize copy and statement, and commit transaction.
	if _, err := stmt.Exec(); err != nil {
		log.Fatal(err)
	}
	if err := stmt.Close(); err != nil {
		log.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

	// Query rows to verify.
	rows, err := db.Query(`select * from users order by name`)
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var (
			name string
			age  int
		)
		err := rows.Scan(&name, &age)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(name, age)
	}

	// Output:
	// Donald Duck 36
	// Scrooge McDuck 86
}

func ExampleNewConnector() {
	c, err := pq.NewConnector("host=postgres dbname=pqgo")
	if err != nil {
		log.Fatal(err)
	}

	db := sql.OpenDB(c)
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}

func ExampleConnectorWithNoticeHandler() {
	// Base connector to wrap
	dsn := ""
	base, err := pq.NewConnector(dsn)
	if err != nil {
		log.Fatal(err)
	}

	// Wrap the connector to simply print out the message
	connector := pq.ConnectorWithNoticeHandler(base, func(notice *pq.Error) {
		fmt.Println("Notice sent: " + notice.Message)
	})
	db := sql.OpenDB(connector)
	defer db.Close()

	// Raise a notice
	sql := "DO language plpgsql $$ BEGIN RAISE NOTICE 'test notice'; END $$"
	if _, err := db.Exec(sql); err != nil {
		log.Fatal(err)
	}
	// Output:
	// Notice sent: test notice
}

func ExampleRegisterTLSConfig() {
	pem, err := os.ReadFile("testdata/init/root.crt")
	if err != nil {
		log.Fatal(err)
	}

	root := x509.NewCertPool()
	root.AppendCertsFromPEM(pem)

	certs, err := tls.LoadX509KeyPair("testdata/init/postgresql.crt", "testdata/init/postgresql.key")
	if err != nil {
		log.Fatal(err)
	}

	pq.RegisterTLSConfig("mytls", &tls.Config{
		RootCAs:      root,
		Certificates: []tls.Certificate{certs},
		ServerName:   "postgres",
	})

	db, err := sql.Open("postgres", "host=postgres dbname=pqgo sslmode=pqgo-mytls")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}
