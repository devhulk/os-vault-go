PROJECT_NAME:="workshop-vault-for-developers"

setup:
	bash scripts/setup.sh
	cp vault-agent/secret-id ../payments-app/agent/secret_id_payments

compose-up:
	docker compose -p $(PROJECT_NAME) -f docker-compose.yml up -d

compose-down:
	docker compose -p $(PROJECT_NAME) -f docker-compose.yml down || true 


agent:
	docker compose -p $(PROJECT_NAME) -f docker-compose-agent.yml up -d

java:
	docker compose -p $(PROJECT_NAME) -f docker-compose-java.yml up -d --build

go:
	docker compose -p $(PROJECT_NAME) -f docker-compose-go.yml up -d --build

clean:
	#docker compose -p $(PROJECT_NAME) -f docker-compose-java.yml down || true
	#docker compose -p $(PROJECT_NAME) -f docker-compose-agent.yml down || true
	docker compose -p $(PROJECT_NAME) -f docker-compose-go.yml down || true
	#rm -rf vault-agent/
	bash scripts/cleanup.sh
