package tools

import (
	"os"
	"strconv"
	"sync"

	"github.com/qdrant/go-client/qdrant"
)

// QdrantClient is a singleton function that returns a Qdrant client
var QdrantClient = sync.OnceValue(func() *qdrant.Client {
	host := os.Getenv("QDRANT_HOST")
	port := os.Getenv("QDRANT_PORT")
	apiKey := os.Getenv("QDRANT_API_KEY")
	if host == "" || port == "" || apiKey == "" {
		panic("QDRANT_HOST, QDRANT_PORT, or QDRANT_API_KEY is not set, please set it in MCP Config")
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		panic("failed to parse QDRANT_PORT: " + err.Error())
	}

	if apiKey == "" {
		panic("QDRANT_API_KEY is not set")
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   host,
		Port:   portInt,
		APIKey: apiKey,
		UseTLS: true,
	})

	if err != nil {
		panic("failed to connect to Qdrant: " + err.Error())
	}

	return client
}) 