run-local:
	@echo "Building Docker image..."
	docker build -t firehose-v1 .
	@echo "Running Docker container..."
	docker run --rm firehose-v1