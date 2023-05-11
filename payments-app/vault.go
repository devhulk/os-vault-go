package main

import (
	"fmt"
	"time"

	"github.com/hashicorp/vault/api"
)

// AuthenticateAppRole - used before implementing vault agent
func AuthenticateAppRole(client *api.Client, roleID, secretID string) (string, error) {
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

// GetDatabaseCredentials - Used throughout to get postgres creds
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

	fmt.Printf("Generated credentials Using SDK: username=%s, password=%s, lease_duration=%s\n", username, password, leaseDuration)

	// TODO: Would need to add logic for live reload here

	return map[string]interface{}{
		"username": username,
		"password": password,
	}, nil
}
