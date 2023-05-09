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
	"time"

	"github.com/hashicorp/vault/api"
)

type Payment struct {
	ID             string    `json:"id,omitempty"`
	Name           string    `json:"name"`
	BillingAddress string    `json:"billing_address"`
	Status         time.Time `json:"status,omitempty"`
}

func GetPayments(db *sql.DB) ([]Payment, error) {
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

func ProcessPayment(vaultClient *api.Client, p Payment) error {
	//posturl := fmt.Sprintf("http://%s/submit", os.Getenv("PAYMENT_PROCESSOR_URL"))
	posturl := fmt.Sprintf("%s/submit", os.Getenv("PAYMENT_PROCESSOR_URL"))
	fmt.Println(posturl)

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

	// Fetch Payment Processor username and password
	processorSecret, err := vaultClient.KVv2("payments/secrets").Get(context.Background(), "processor")
	if err != nil {
		log.Fatalf("unable to read secret: %v", err)
	}

	username, ok := processorSecret.Data["username"].(string)
	if !ok {
		log.Fatalf("value type assertion failed: %T %#v", processorSecret.Data["username"], processorSecret.Data["username"])
	}

	password, ok := processorSecret.Data["password"].(string)
	if !ok {
		log.Fatalf("value type assertion failed: %T %#v", processorSecret.Data["password"], processorSecret.Data["password"])
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Basic "+basicAuth(fmt.Sprintf("%s", username), fmt.Sprintf("%s", password)))

	client := &http.Client{}
	res, err1 := client.Do(r)
	if err1 != nil {
		return err1
	}

	defer res.Body.Close()

	if res.StatusCode != 201 {
		panic(fmt.Sprintf("payment was not processed. Expected 201 and received %v", res.StatusCode))
	}

	return nil

}

func InsertPayment(db *sql.DB, p Payment) (string, error) {

	// Start insert into app database

	ctx := context.Background()

	_, err := db.ExecContext(ctx, `INSERT INTO payments (id, name, billing_address, created_at) VALUES ($1, $2, $3, $4)`,
		p.ID, p.Name, p.BillingAddress, p.Status)

	if err != nil {
		return "", fmt.Errorf("failed to insert payment: %v", err)
	}

	return "success", nil
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
