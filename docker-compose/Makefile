PROJECT_NAME:="workshop-vault-for-developers"

setup:
	bash scripts/setup.sh
	cp vault-agent/secret-id ../payments-app/agent/secret_id_payments

v-agent:
	docker compose -p $(PROJECT_NAME) -f docker-compose-agent.yml up -d

go:
	docker compose -p $(PROJECT_NAME) -f docker-compose-go.yml up -d --build

compose-down:
	docker compose -p $(PROJECT_NAME) -f docker-compose.yml down || true 


java:
	docker compose -p $(PROJECT_NAME) -f docker-compose-java.yml up -d --build


clean:
	#docker compose -p $(PROJECT_NAME) -f docker-compose-java.yml down || true
	#docker compose -p $(PROJECT_NAME) -f docker-compose-agent.yml down || true
	docker compose -p $(PROJECT_NAME) -f docker-compose-go.yml down || true
	#rm -rf vault-agent/
	bash scripts/cleanup.sh
