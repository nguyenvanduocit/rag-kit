package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/qdrant/go-client/qdrant"
)

// CreateCollectionTool creates a new collection handler
func CreateCollectionTool() mcp.Tool {
	return mcp.NewTool("memory_create_collection",
		mcp.WithDescription("Create a new vector collection in memory"),
		mcp.WithString("collection", mcp.Required(), mcp.Description("Memory collection name")),
	)
}

// CreateCollectionHandler handles the creation of a new collection
func CreateCollectionHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	collection := request.Params.Arguments["collection"].(string)

	// Check if collection already exists
	collectionInfo, err := QdrantClient().GetCollectionInfo(ctx, collection)
	if err == nil && collectionInfo != nil {
		return nil, fmt.Errorf("collection %s already exists", collection)
	}

	// Create collection with configuration for text embeddings
	err = QdrantClient().CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collection,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     uint64(2048), // OpenAI text-embedding-3-large dimension
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %v", err)
	}

	result := fmt.Sprintf("Successfully created collection: %s", collection)
	return mcp.NewToolResultText(result), nil
}

// DeleteCollectionTool creates a delete collection handler
func DeleteCollectionTool() mcp.Tool {
	return mcp.NewTool("memory_delete_collection",
		mcp.WithDescription("Delete a vector collection in memory"),
		mcp.WithString("collection", mcp.Required(), mcp.Description("Memory collection name")),
	)
}

// DeleteCollectionHandler handles the deletion of a collection
func DeleteCollectionHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	collection := request.Params.Arguments["collection"].(string)

	// Check if collection exists
	collectionInfo, err := QdrantClient().GetCollectionInfo(ctx, collection)
	if err != nil || collectionInfo == nil {
		return nil, fmt.Errorf("collection %s does not exist", collection)
	}

	// Delete collection
	err = QdrantClient().DeleteCollection(ctx, collection)
	if err != nil {
		return nil, fmt.Errorf("failed to delete collection: %v", err)
	}

	result := fmt.Sprintf("Successfully deleted collection: %s", collection)
	return mcp.NewToolResultText(result), nil
}

// ListCollectionsTool creates a list collections handler
func ListCollectionsTool() mcp.Tool {
	return mcp.NewTool("memory_list_collections",
		mcp.WithDescription("List all vector collections in memory"),
	)
}

// ListCollectionsHandler handles listing all collections
func ListCollectionsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	collections, err := QdrantClient().ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	return mcp.NewToolResultText(fmt.Sprintf("Collections: %v", collections)), nil
} 