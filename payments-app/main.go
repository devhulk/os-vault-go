package main

import (
	"database/sql"
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
	creds, err := GetDatabaseCredentials(vaultClient, vaultDBRole)
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

	// Read new token file
	r.GET("/refresh-token", func(ctx *gin.Context) {

		vaultToken, err := os.ReadFile("./agent/client-token")
		if err != nil {
			log.Fatalf("Failed to read token file. Error: %s", err)
		}

		vaultClient.SetToken(string(vaultToken))

		ctx.JSON(200, gin.H{
			"message": "token successfully refreshed.",
		})

	})

	r.GET("/payments", func(ctx *gin.Context) {
		p, err := GetPayments(db)

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

		err := ProcessPayment(vaultClient, p)
		if err != nil {
			ctx.JSON(400, gin.H{
				"error": err,
			})

		}

		status, err := InsertPayment(db, p)

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
