package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nguyenvanduocit/rag-kit/services"
	"github.com/qdrant/go-client/qdrant"
	"github.com/sashabaranov/go-openai"
)

// SearchTool creates a tool for vector search
func SearchTool() mcp.Tool {
	return mcp.NewTool("memory_search",
		mcp.WithDescription("Search for memory in a collection based on a query"),
		mcp.WithString("collection", mcp.Required(), mcp.Description("Memory collection name")),
		mcp.WithString("query", mcp.Required(), mcp.Description("search query, should be a keyword")),
	)
}

// SearchHandler handles vector searches
func SearchHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	collection := request.Params.Arguments["collection"].(string)
	query := request.Params.Arguments["query"].(string)
	
	// Generate embedding for the query
	resp, err := services.DefaultOpenAIClient().CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input:      []string{query},
		Model:      openai.LargeEmbedding3,
		Dimensions: 2048,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings for query: %v", err)
	}

	scoreThreshold := float32(0.6)
	// Search Qdrant
	searchResult, err := qdrantClient().Query(context.Background(), &qdrant.QueryPoints{
		CollectionName: collection,
		Query:          qdrant.NewQuery(resp.Data[0].Embedding...),
		ScoreThreshold: &scoreThreshold,
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search in Qdrant: %v", err)
	}

	// Format results
	var resultText string
	for i, hit := range searchResult {
		content := hit.Payload["content"].GetStringValue()
		filePath := hit.Payload["filePath"].GetStringValue()
		resultText += fmt.Sprintf("Result %d (Score: %f):\nFilePath: %s\nContent: %s\n\n", i+1, hit.Score, filePath, content)
	}

	return mcp.NewToolResultText(resultText), nil
} 