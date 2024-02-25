build:
	@echo "Building Docker image..."
	docker build -t firehose-v1 .
run-local: build
	@echo "Running Docker container..."
	docker run --rm --env-file .env -p 7878:8000 firehose-v1