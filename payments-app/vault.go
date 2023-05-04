package main

import (
	"fmt"
	"time"

	"github.com/hashicorp/vault/api"
)

func GetDatabaseCredentials(client *api.Client, roleName string) (map[string]interface{}, error) {
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
