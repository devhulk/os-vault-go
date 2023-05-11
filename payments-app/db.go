package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/hashicorp/vault/api"
)

// Repo
type Repo struct {
	Creds       map[string]interface{}
	VaultClient *api.Client
	DB          *sql.DB
}

func InitDB(vaultClient *api.Client, vaultDBRole string, postgresDBURL string) (*Repo, error) {
	// Init DB Connection
	creds, err := GetDatabaseCredentials(vaultClient, vaultDBRole)
	if err != nil {
		log.Fatalf("Failed to get database credentials: %s", err)
	}
	// Connect to the PostgreSQL database using the fetched credentials
	db, err := sql.Open("postgres", fmt.Sprintf("user=%s password=%s dbname=%s host=%s sslmode=disable", creds["username"], creds["password"], "payments", postgresDBURL))
	if err != nil {
		log.Fatalf("Insided INIT Function: Failed to connect to the database: %s", err)
	}

	defer db.Close()

	// Check database connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping the database: %s", err)
	}
	return &Repo{
		Creds:       creds,
		VaultClient: vaultClient,
		DB:          db,
	}, nil
}

func (r Repo) RefreshCredentials(vaultClient *api.Client, vaultDBRole string, postgresDBURL string) {

	r.DB.Close()

	creds, err := GetDatabaseCredentials(vaultClient, vaultDBRole)
	if err != nil {
		log.Fatalf("Failed to get database credentials: %s", err)
	}

	r.Creds = creds

	db, err := sql.Open("postgres", fmt.Sprintf("user=%s password=%s dbname=%s host=%s sslmode=disable", creds["username"], creds["password"], "payments", postgresDBURL))
	if err != nil {
		log.Fatalf("Failed to connect to the database: %s", err)
	}

	defer db.Close()

	r.DB = db

}
