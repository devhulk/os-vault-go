package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hashicorp/vault/api"
	_ "github.com/lib/pq"
)

// TODO: Handle refresh of credentials. Reload creds file.

type Config struct {
	ProcessorUsername *string
	ProcessorPassword *string
	DatabaseUsername  *string
	DatabasePassword  *string
	DB                *sql.DB
}

func scanDBConfig(file *os.File, config *Config) {

	scanner := bufio.NewScanner(file)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		txt := scanner.Text()
		//fmt.Println(scanner.Text())
		if len(txt) > 0 {
			split := strings.Split(txt, "=")
			//fmt.Println("username: ", split)
			switch value := split[0]; value {
			case "username":
				fmt.Println("username: ", split[1])
				config.DatabaseUsername = &split[1]
			case "password":
				fmt.Println("password: ", split[1])
				config.DatabasePassword = &split[1]
			case "url":
				fmt.Println("url: ", split[1])
			}
		}

	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func scanProcessorConfig(file *os.File, config *Config) {

	scanner := bufio.NewScanner(file)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		txt := scanner.Text()
		//fmt.Println(scanner.Text())
		if len(txt) > 0 {
			split := strings.Split(txt, "=")
			//fmt.Println("username: ", split)
			switch value := split[0]; value {
			case "username":
				fmt.Println("username: ", split[1])
				config.ProcessorUsername = &split[1]
			case "password":
				fmt.Println("password: ", split[1])
				config.ProcessorPassword = &split[1]
			case "url":
				fmt.Println("url: ", split[1])
			}
		}

	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func main() {

	var config Config

	config = Config{}

	vaultAddr := os.Getenv("VAULT_ADDR")
	vaultDBRole := os.Getenv("VAULT_DB_ROLE")
	postgresDBURL := os.Getenv("POSTGRES_DB_URL")
	clientTokenFile := os.Getenv("VAULT_TOKEN_FILE")

	//export DB_CONFIG_FILE="../../docker-compose/vault/secrets/database.properties"
	//export PROCESSOR_CONFIG_FILE="../../docker-compose/vault/secrets/processor.properties"

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
	vaultToken, err := os.ReadFile(clientTokenFile)
	if err != nil {
		log.Fatalf("Failed to authenticate via APP ROLE with Vault: %s", err)
	}

	dbFile, err := os.ReadFile("/vault/secrets/database.properties")
	if err != nil {
		log.Fatalf("Failed to authenticate via APP ROLE with Vault: %s", err)
	}

	fmt.Println(string(dbFile))

	// TODO:
	fmt.Println(string(vaultToken))

	vaultClient.SetToken(string(vaultToken))
	// Fetch credentials from Vault using role and Vault Agent provided token
	creds, err := GetDatabaseCredentials(vaultClient, vaultDBRole)
	if err != nil {
		log.Fatalf("Failed to get database credentials: %s", err)
	}

	connectionString := fmt.Sprintf("user=%s password=%s dbname=%s host=%s sslmode=disable", creds["username"], creds["password"], "payments", postgresDBURL)

	fmt.Println(connectionString)
	// Connect to the PostgreSQL database using the fetched credentials
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %s", err)
	}

	config.DB = db

	defer db.Close()

	// Check database connection
	//err = db.Ping()
	//if err != nil {
	//log.Fatalf("Failed to ping the database: %s", err)
	//}

	r := gin.Default()

	r.GET("/health-check", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message": "service is up and healthy",
		})

	})

	// Read new token file
	r.GET("/reload", func(ctx *gin.Context) {

		//var dbUserName, dbPassword string
		dbFile, err := os.Open("/vault/secrets/database.properties")
		if err != nil {
			log.Fatalf("Failed to authenticate via APP ROLE with Vault: %s", err)
		}

		defer dbFile.Close()

		scanDBConfig(dbFile, &config)

		connectionString := fmt.Sprintf("user=%s password=%s dbname=%s host=%s sslmode=disable", *config.DatabaseUsername, *config.DatabasePassword, "payments", postgresDBURL)

		fmt.Println(connectionString)
		// Connect to the PostgreSQL database using the fetched credentials
		db, err := sql.Open("postgres", connectionString)
		if err != nil {
			log.Fatalf("Failed to connect to the database: %s", err)
		}

		config.DB = db

		processorFile, err := os.Open("/vault/secrets/processor.properties")
		if err != nil {
			log.Fatalf("Failed to authenticate via APP ROLE with Vault: %s", err)
		}

		defer processorFile.Close()

		scanProcessorConfig(processorFile, &config)

		vaultToken, err := os.ReadFile(clientTokenFile)
		if err != nil {
			log.Fatalf("Failed to read token file. Error: %s", err)
		}

		vaultClient.SetToken(string(vaultToken))

		fmt.Println("This is the Global Config: ")
		fmt.Println("u :", *config.DatabaseUsername)
		fmt.Println("p :", *config.DatabasePassword)
		fmt.Println("u :", *config.ProcessorUsername)
		fmt.Println("p :", *config.ProcessorPassword)

		//fmt.Println("Reloaded DB Config: ", string(dbFile))
		//fmt.Println("Reloaded Processor Config: ", string(processorFile))
		//fmt.Println("Reloaded Client Token: ", string(vaultToken))

		ctx.JSON(200, gin.H{
			"message": "token successfully refreshed.",
		})

	})

	r.GET("/payments", func(ctx *gin.Context) {
		//fmt.Println("Creds for /payments", creds["username"], creds["password"])
		p, err := GetPayments(&config)

		fmt.Println(*config.DatabaseUsername)
		fmt.Println(*config.DatabasePassword)

		if err != nil {
			log.Println(err)
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
			log.Println("Processing Error: ", err)

		}

		status, err := InsertPayment(db, p)
		if err != nil {
			log.Println("Processing Error: ", err)
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
