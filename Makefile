.PHONY: test test-gateway test-processor test-dashboard up down build lint

test: test-gateway test-processor test-dashboard
	@echo "All tests passed!"

test-gateway:
	cd services/gateway && pip install -q -r requirements.txt && pytest -v

test-processor:
	cd services/processor && go test -v ./...

test-dashboard:
	cd services/dashboard && npm install --silent && npm test

lint: lint-gateway lint-processor lint-dashboard
	@echo "All linters passed!"

lint-gateway:
	cd services/gateway && flake8 --max-line-length=100 app.py

lint-processor:
	cd services/processor && go vet ./...

lint-dashboard:
	cd services/dashboard && npm install --silent && npx eslint src/

build:
	docker compose build

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

restart:
	docker compose restart

status:
	docker compose ps
