ENV_FILE := .env
ENV := $(shell cat $(ENV_FILE))

.PHONY:run
run:
	$(ENV) go run cmd/xss-demo/main.go

.PHONY:docker/up
docker/up:
	docker compose -f docker-compose.yml up --build -d

.PHONY:docker/down
docker/down:
	docker compose -f docker-compose.yml down

.PHONY:migrate
migrate:
	docker exec -it db bash /tmp/init_database.sh

.PHONY:log
log:
	docker compose logs -f
