build:
	@echo "Building Docker image..."
	docker build -t firehose-v1 .
run: build
	@echo "Running Docker container..."
	docker run --rm --env-file .env -p 7878:8000 firehose-v1

run-local:
	@echo "Running Go run"
	go run .
