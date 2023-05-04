package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hashicorp/vault/api"
	_ "github.com/lib/pq"
)

type Payment struct {
	ID             string    `json:"id,omitempty"`
	Name           string    `json:"name"`
	BillingAddress string    `json:"billing_address"`
	Status         time.Time `json:"status,omitempty"`
}

func main() {
	vaultAddr := os.Getenv("VAULT_ADDR")
	vaultDBRole := os.Getenv("VAULT_DB_ROLE")

	if vaultAddr == "" || vaultDBRole == "" {
		log.Fatalf("Environment variables VAULT_ADDR, VAULT_DB_ROLE must be set.")
	}

	// Initialize the Vault client
	vaultConfig := &api.Config{
		Address: vaultAddr,
	}

	vaultClient, err := api.NewClient(vaultConfig)
	if err != nil {
		log.Fatalf("Failed to create Vault client: %s", err)
	}

	// Auth using client-token from Vault Agent
	vaultToken, err := os.ReadFile("./agent/client-token")
	if err != nil {
		log.Fatalf("Failed to authenticate via APP ROLE with Vault: %s", err)
	}

	vaultClient.SetToken(string(vaultToken))

	// Fetch credentials from Vault using role and Vault Agent provided token
	creds, err := getDatabaseCredentials(vaultClient, vaultDBRole)
	if err != nil {
		log.Fatalf("Failed to get database credentials: %s", err)
	}

	// Connect to the PostgreSQL database using the fetched credentials
	db, err := sql.Open("postgres", fmt.Sprintf("user=%s password=%s dbname=%s host=localhost sslmode=disable", creds["username"], creds["password"], "payments"))
	if err != nil {
		log.Fatalf("Failed to connect to the database: %s", err)
	}

	defer db.Close()

	// Check database connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping the database: %s", err)
	}

	r := gin.Default()

	r.GET("/payments", func(ctx *gin.Context) {
		p, err := getPayments(db)

		fmt.Println(p)

		if err != nil {
			ctx.JSON(400, gin.H{
				"error": "could not get records from database.",
			})
		}

		ctx.JSON(200, p)

	})

	r.GET("/payments/:id", func(ctx *gin.Context) {
		id := ctx.Param("id")
		fmt.Println(id)

	})

	r.POST("/payments", func(ctx *gin.Context) {

		var p Payment

		p.ID = uuid.New().String()
		p.Status = time.Now()

		if err := ctx.BindJSON(&p); err != nil {
			return
		}

		err := processPayment(p)
		if err != nil {
			ctx.JSON(400, gin.H{
				"error": err,
			})

		}

		status, err := insertPayment(db, p)

		if err != nil {
			fmt.Println(err)
			ctx.JSON(400, gin.H{
				"error": err,
			})
		}

		ctx.JSON(200, gin.H{
			"message": fmt.Sprintf("Payment Status: %s", status),
		})

	})

	srv := &http.Server{
		Addr:    ":8081",
		Handler: r,
	}

	// service connections
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Listen for interrupt signal (Ctrl+C) and gracefully disconnect from the database
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalChan

		fmt.Println("\nReceived an interrupt signal, gracefully shutting down...")

		err := db.Close()
		if err != nil {
			log.Printf("Failed to close database connection: %s", err)
		} else {
			fmt.Println("Disconnected from the db.")
		}

		fmt.Println("Gracefully shutting down.")

		os.Exit(0)
	}()

	// Keep the application running
	select {}
}

func getPayments(db *sql.DB) ([]Payment, error) {
	// Prepare the SQL query
	query := "SELECT * FROM payments"
	var payments []Payment

	// Execute the query and get the results
	rows, err := db.Query(query)
	if err != nil {
		// Handle error
		fmt.Printf("ERROR: Could not execute query. \n %v", err)
		return nil, err
	}
	defer rows.Close()

	// Iterate over the results and print them out
	for rows.Next() {
		payment := Payment{}

		err := rows.Scan(&payment.ID, &payment.Name, &payment.BillingAddress, &payment.Status)
		if err != nil {
			fmt.Printf("ERROR: Could not scan rows. \n %v", err)
			return nil, err
		}

		fmt.Println("pre append", payments)

		payments = append(payments, payment)

		fmt.Printf("Payment ID: %s, Customer Name: %s, Billing Address: %s, Created At: %s \n", payment.ID, payment.Name, payment.BillingAddress, payment.Status)

	}

	if err := rows.Err(); err != nil {
		fmt.Println("ERROR: Could not get rows.")
		return nil, err
	}

	return payments, nil

}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func processPayment(p Payment) error {
	// TODO: Get username and password from vault for Payment Processor (kv2)

	posturl := "http://localhost:8080/submit"

	// value encryption**
	bEncoded := base64.StdEncoding.EncodeToString([]byte(p.BillingAddress))

	body := []byte(fmt.Sprintf(`{
			"name": "%s",
			"billing_address": "%s"
			}
	`, p.Name, bEncoded))

	// Create a HTTP post request
	r, e := http.NewRequest("POST", posturl, bytes.NewBuffer(body))
	if e != nil {
		return e
	}

	r.Header.Add("Content-Type", "application/json")
	// TODO: Add vault or env vars -> otherwise returns 401
	r.Header.Add("Authorization", "Basic "+basicAuth("", ""))

	client := &http.Client{}
	res, err1 := client.Do(r)
	if err1 != nil {
		return err1
	}

	defer res.Body.Close()

	//b, errd := io.ReadAll(res.Body)
	//if errd != nil {
	//return errd
	//}

	//fmt.Println(string(b))

	//fmt.Println(res.StatusCode)

	if res.StatusCode != 201 {
		panic(fmt.Sprintf("payment was not processed. Expected 201 and received %v", res.StatusCode))
	}

	return nil

}

func insertPayment(db *sql.DB, p Payment) (string, error) {

	// Start insert into app database

	ctx := context.Background()

	_, err := db.ExecContext(ctx, `INSERT INTO payments (id, name, billing_address, created_at) VALUES ($1, $2, $3, $4)`,
		p.ID, p.Name, p.BillingAddress, p.Status)

	if err != nil {
		return "", fmt.Errorf("failed to insert payment: %v", err)
	}

	// TODO : replace with payment processor status message
	return "success", nil
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
	secret, err := client.Logical().Read(fmt.Sprintf("payments/database/creds/%s", roleName))

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
