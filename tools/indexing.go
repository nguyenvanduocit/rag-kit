package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nguyenvanduocit/rag-kit/services"
	"github.com/qdrant/go-client/qdrant"
	"github.com/sashabaranov/go-openai"
)

// IndexContentTool creates a tool for indexing content
func IndexContentTool() mcp.Tool {
	return mcp.NewTool("memory_index_content",
		mcp.WithDescription("Index a content into memory, can be inserted or updated"),
		mcp.WithString("collection", mcp.Required(), mcp.Description("Memory collection name")),
		mcp.WithString("filePath", mcp.Required(), mcp.Description("content file path")),
		mcp.WithString("payload", mcp.Required(), mcp.Description("Plain text payload")),
	)
}

// IndexContentHandler handles the indexing of content
func IndexContentHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	collection := request.Params.Arguments["collection"].(string)
	filePath := request.Params.Arguments["filePath"].(string)
	payload := request.Params.Arguments["payload"].(string)

	// Split content into chunks
	chunks, err := SplitIntoChunks(payload, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to split into chunks: %v", err)
	}

	var points []*qdrant.PointStruct
	for i, chunk := range chunks {
		// Generate embeddings for each chunk
		resp, err := services.DefaultOpenAIClient().CreateEmbeddings(context.Background(), openai.EmbeddingRequest{
			Input:      []string{chunk},
			Model:      openai.LargeEmbedding3,
			Dimensions: 2048,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to generate embeddings: %v", err)
		}

		// Create point for each chunk
		point := &qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(uuid.NewSHA1(uuid.NameSpaceURL, []byte(filePath+strconv.Itoa(i))).String()),
			Vectors: qdrant.NewVectors(resp.Data[0].Embedding...),
			Payload: qdrant.NewValueMap(map[string]any{
				"filePath":   filePath,
				"content":    chunk,
				"chunkIndex": i,
			}),
		}
		points = append(points, point)
	}

	waitUpsert := true

	// Upsert all chunks
	upsertResp, err := QdrantClient().Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collection,
		Wait:           &waitUpsert,
		Points:         points,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert points: %v", err)
	}

	result := fmt.Sprintf("Successfully upserted\nOperation ID: %d\nStatus: %s", upsertResp.OperationId, upsertResp.Status)

	return mcp.NewToolResultText(result), nil
}

// DeleteIndexByFilePathTool creates a tool for deleting indexes by file path
func DeleteIndexByFilePathTool() mcp.Tool {
	return mcp.NewTool("memory_delete_index_by_filepath",
		mcp.WithDescription("Delete a vector index by filePath"),
		mcp.WithString("collection", mcp.Required(), mcp.Description("Memory collection name")),
		mcp.WithString("filePath", mcp.Required(), mcp.Description("Path to the local file to be deleted")),
	)
}

// DeleteIndexByFilePathHandler handles the deletion of indexes by file path
func DeleteIndexByFilePathHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	collection := request.Params.Arguments["collection"].(string)
	filePath := request.Params.Arguments["filePath"].(string)

	// Delete points by IDs using PointSelector
	pointsSelector := &qdrant.PointsSelector{
		PointsSelectorOneOf: &qdrant.PointsSelector_Filter{
			Filter: &qdrant.Filter{
				Must: []*qdrant.Condition{
					{
						ConditionOneOf: &qdrant.Condition_Field{
							Field: &qdrant.FieldCondition{
								Key: "filePath",
								Match: &qdrant.Match{
									MatchValue: &qdrant.Match_Text{
										Text: filePath,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	deleteResp, err := QdrantClient().Delete(ctx, &qdrant.DeletePoints{
		CollectionName: collection,
		Points:         pointsSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to delete points for filePath %s: %v", filePath, err)
	}

	result := fmt.Sprintf("Successfully deleted points for filePath: %s\nOperation ID: %d\nStatus: %s", filePath, deleteResp.OperationId, deleteResp.Status)
	return mcp.NewToolResultText(result), nil
} 