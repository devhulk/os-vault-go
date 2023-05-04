
# Order of Testing

1. Trusted entity introduces Vault-Agent
2. Vault Agent auto renews auth token for app.
3. Vault Agent writes Database Creds to File

# Testing 

Get db creds

```
vault read payments/database/creds/payments-app
```

# Presentation Workflow

1. Explain Application Architecture and Workflow (Gin, Postgres, Payment Processor)
2. Using App Role and the Go SDK
3. Off loading token management to Vault Agent - Explain scenarios where you would use both.
4. Deploy Using Vault Agent and k8s

# Running the Project

```
go run *.go
```

or build and run **os-vault** binary 

```
go build
./os-vault
```

