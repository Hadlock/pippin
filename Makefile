.PHONY: run build test docker clean

run:
	ACCOUNT_ID=demo SPRINT_LENGTH_DAYS=7 go run main.go

build:
	go build -o pippin main.go

test:
	curl -s http://localhost:8080/api/projects | jq .
	curl -s http://localhost:8080/api/tickets | jq .

docker:
	docker build -t pippin:latest .

clean:
	rm -f pippin pippin.db
