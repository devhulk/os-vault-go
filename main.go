package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/vault/api"
	_ "github.com/lib/pq"
)

func main() {
	vaultAddr := os.Getenv("VAULT_ADDR")
	vaultRoleID := os.Getenv("VAULT_ROLE_ID")
	vaultSecretID := os.Getenv("VAULT_SECRET_ID")
	vaultDBRole := os.Getenv("VAULT_DB_ROLE")

	if vaultAddr == "" || vaultRoleID == "" || vaultSecretID == "" || vaultDBRole == "" {
		log.Fatalf("Environment variables VAULT_ADDR, VAULT_ROLE_ID, VAULT_SECRET_ID, and VAULT_DB_ROLE must be set.")
	}

	// Initialize the Vault client
	vaultConfig := &api.Config{
		Address: vaultAddr,
	}

	vaultClient, err := api.NewClient(vaultConfig)
	if err != nil {
		log.Fatalf("Failed to create Vault client: %s", err)
	}

	// Authenticate with Vault using AppRole
	vaultToken, err := authenticateAppRole(vaultClient, vaultRoleID, vaultSecretID)
	if err != nil {
		log.Fatalf("Failed to authenticate with Vault: %s", err)
	}
	vaultClient.SetToken(vaultToken)

	// Fetch credentials from Vault
	creds, err := getDatabaseCredentials(vaultClient, vaultDBRole)
	if err != nil {
		log.Fatalf("Failed to get database credentials: %s", err)
	}

	// Connect to the PostgreSQL database using the fetched credentials
	db, err := sql.Open("postgres", fmt.Sprintf("user=%s password=%s dbname=mydb host=localhost sslmode=disable", creds["username"], creds["password"]))
	if err != nil {
		log.Fatalf("Failed to connect to the database: %s", err)
	}
	defer db.Close()

	// Check database connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping the database: %s", err)
	}

	fmt.Println("Connected to the PostgreSQL database using Vault-generated credentials!")

	// Listen for interrupt signal (Ctrl+C) and gracefully disconnect from the database
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChan
		fmt.Println("\nReceived an interrupt signal, disconnecting from the database...")
		err := db.Close()
		if err != nil {
			log.Printf("Failed to close database connection: %s", err)
		} else {
			fmt.Println("Disconnected from the database.")
		}
		os.Exit(0)
	}()

	// Keep the application running
	select {}
}

func authenticateAppRole(client *api.Client, roleID, secretID string) (string, error) {
	// Authenticate using the role_id and secret_id
	data := map[string]interface{}{
		"role_id":   roleID,
		"secret_id": secretID,
	}

	secret, err := client.Logical().Write("auth/approle/login", data)
	if err != nil {
		return "", err
	}
	if secret == nil || secret.Auth == nil {
		return "", fmt.Errorf("failed to authenticate with approle")
	}

	return secret.Auth.ClientToken, nil
}
func getDatabaseCredentials(client *api.Client, roleName string) (map[string]interface{}, error) {
	// Generate a new set of credentials by reading from the Vault role
	secret, err := client.Logical().Read(fmt.Sprintf("database/creds/%s", roleName))
	if err != nil {
		return nil, err
	}
	if secret == nil {
		return nil, fmt.Errorf("no credentials found for role %s", roleName)
	}

	// Extract the username and password from the secret
	username := secret.Data["username"].(string)
	password := secret.Data["password"].(string)
	leaseDuration := time.Duration(secret.LeaseDuration) * time.Second

	fmt.Printf("Generated credentials: username=%s, password=%s, lease_duration=%s\n", username, password, leaseDuration)

	return map[string]interface{}{
		"username": username,
		"password": password,
	}, nil
}
