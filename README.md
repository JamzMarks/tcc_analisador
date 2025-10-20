Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass

# run.ps1
$env:AMQP_URL="amqp://user:pass@host.docker.internal:5672/"
$env:QUEUE_NAME="test-queue"
$env:GRAPH_DB_URI="bolt://host.docker.internal:7687"
$env:GRAPH_DB_USERNAME="neo4j"
$env:GRAPH_DB_PASSWORD="neo4j"

go run main.go

