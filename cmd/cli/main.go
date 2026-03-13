package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/nguyenvanduocit/rag-kit/services"
	"github.com/nguyenvanduocit/rag-kit/tools"
	"github.com/qdrant/go-client/qdrant"
	"github.com/sashabaranov/go-openai"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "create-collection":
		runCreateCollection(os.Args[2:])
	case "delete-collection":
		runDeleteCollection(os.Args[2:])
	case "list-collections":
		runListCollections(os.Args[2:])
	case "index-content":
		runIndexContent(os.Args[2:])
	case "delete-index":
		runDeleteIndex(os.Args[2:])
	case "search":
		runSearch(os.Args[2:])
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`rag-cli - RAG kit command line interface

Usage:
  rag-cli <command> [flags]

Commands:
  create-collection   Create a new vector collection
  delete-collection   Delete an existing vector collection
  list-collections    List all vector collections
  index-content       Index content into a collection
  delete-index        Delete indexed content by file path
  search              Search for content in a collection

Global Flags:
  --env string      Path to .env file (optional)
  --output string   Output format: text (default) or json

Required Environment Variables:
  OPENAI_API_KEY    OpenAI API key
  QDRANT_HOST       Qdrant server host
  QDRANT_PORT       Qdrant server port
  QDRANT_API_KEY    Qdrant API key

Run 'rag-cli <command> --help' for command-specific flags.
`)
}

func loadEnv(envFile string) {
	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			fmt.Fprintf(os.Stderr, "error loading env file %s: %v\n", envFile, err)
			os.Exit(1)
		}
	}
}

func checkEnv() {
	missing := []string{}
	for _, v := range []string{"OPENAI_API_KEY", "QDRANT_HOST", "QDRANT_PORT", "QDRANT_API_KEY"} {
		if os.Getenv(v) == "" {
			missing = append(missing, v)
		}
	}
	if len(missing) > 0 {
		for _, v := range missing {
			fmt.Fprintf(os.Stderr, "error: required environment variable %s is not set\n", v)
		}
		os.Exit(1)
	}
}

func outputResult(format string, textResult string, jsonData any) {
	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(jsonData); err != nil {
			fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println(textResult)
	}
}

func newQdrantClient() *qdrant.Client {
	host := os.Getenv("QDRANT_HOST")
	port := os.Getenv("QDRANT_PORT")
	apiKey := os.Getenv("QDRANT_API_KEY")

	portInt, err := strconv.Atoi(port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid QDRANT_PORT value %q: %v\n", port, err)
		os.Exit(1)
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   host,
		Port:   portInt,
		APIKey: apiKey,
		UseTLS: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error connecting to Qdrant: %v\n", err)
		os.Exit(1)
	}
	return client
}

// ---- create-collection ----

func runCreateCollection(args []string) {
	fs := flag.NewFlagSet("create-collection", flag.ExitOnError)
	envFile := fs.String("env", "", "path to .env file")
	output := fs.String("output", "text", "output format: text or json")
	collection := fs.String("collection", "", "collection name (required)")
	fs.Parse(args)

	loadEnv(*envFile)
	checkEnv()

	if *collection == "" {
		fmt.Fprintln(os.Stderr, "error: --collection is required")
		fs.Usage()
		os.Exit(1)
	}

	ctx := context.Background()
	client := newQdrantClient()

	info, err := client.GetCollectionInfo(ctx, *collection)
	if err == nil && info != nil {
		fmt.Fprintf(os.Stderr, "error: collection %s already exists\n", *collection)
		os.Exit(1)
	}

	err = client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: *collection,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     uint64(2048),
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to create collection: %v\n", err)
		os.Exit(1)
	}

	text := fmt.Sprintf("Successfully created collection: %s", *collection)
	outputResult(*output, text, map[string]string{"collection": *collection, "status": "created"})
}

// ---- delete-collection ----

func runDeleteCollection(args []string) {
	fs := flag.NewFlagSet("delete-collection", flag.ExitOnError)
	envFile := fs.String("env", "", "path to .env file")
	output := fs.String("output", "text", "output format: text or json")
	collection := fs.String("collection", "", "collection name (required)")
	fs.Parse(args)

	loadEnv(*envFile)
	checkEnv()

	if *collection == "" {
		fmt.Fprintln(os.Stderr, "error: --collection is required")
		fs.Usage()
		os.Exit(1)
	}

	ctx := context.Background()
	client := newQdrantClient()

	info, err := client.GetCollectionInfo(ctx, *collection)
	if err != nil || info == nil {
		fmt.Fprintf(os.Stderr, "error: collection %s does not exist\n", *collection)
		os.Exit(1)
	}

	err = client.DeleteCollection(ctx, *collection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to delete collection: %v\n", err)
		os.Exit(1)
	}

	text := fmt.Sprintf("Successfully deleted collection: %s", *collection)
	outputResult(*output, text, map[string]string{"collection": *collection, "status": "deleted"})
}

// ---- list-collections ----

func runListCollections(args []string) {
	fs := flag.NewFlagSet("list-collections", flag.ExitOnError)
	envFile := fs.String("env", "", "path to .env file")
	output := fs.String("output", "text", "output format: text or json")
	fs.Parse(args)

	loadEnv(*envFile)
	checkEnv()

	ctx := context.Background()
	client := newQdrantClient()

	collections, err := client.ListCollections(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to list collections: %v\n", err)
		os.Exit(1)
	}

	text := fmt.Sprintf("Collections: %v", collections)
	outputResult(*output, text, map[string]any{"collections": collections})
}

// ---- index-content ----

func runIndexContent(args []string) {
	fs := flag.NewFlagSet("index-content", flag.ExitOnError)
	envFile := fs.String("env", "", "path to .env file")
	output := fs.String("output", "text", "output format: text or json")
	collection := fs.String("collection", "", "collection name (required)")
	filePath := fs.String("file-path", "", "content file path identifier (required)")
	payload := fs.String("payload", "", "plain text payload to index (required)")
	fs.Parse(args)

	loadEnv(*envFile)
	checkEnv()

	if *collection == "" || *filePath == "" || *payload == "" {
		fmt.Fprintln(os.Stderr, "error: --collection, --file-path, and --payload are required")
		fs.Usage()
		os.Exit(1)
	}

	ctx := context.Background()

	chunks, err := tools.SplitIntoChunks(*payload, *filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to split into chunks: %v\n", err)
		os.Exit(1)
	}

	var points []*qdrant.PointStruct
	for i, chunk := range chunks {
		resp, err := services.DefaultOpenAIClient().CreateEmbeddings(ctx, openai.EmbeddingRequest{
			Input:      []string{chunk},
			Model:      openai.LargeEmbedding3,
			Dimensions: 2048,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to generate embeddings for chunk %d: %v\n", i, err)
			os.Exit(1)
		}

		pointID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(*filePath+strconv.Itoa(i))).String()
		point := &qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(pointID),
			Vectors: qdrant.NewVectors(resp.Data[0].Embedding...),
			Payload: qdrant.NewValueMap(map[string]any{
				"filePath":   *filePath,
				"content":    chunk,
				"chunkIndex": i,
			}),
		}
		points = append(points, point)
	}

	client := newQdrantClient()
	waitUpsert := true
	upsertResp, err := client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: *collection,
		Wait:           &waitUpsert,
		Points:         points,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to upsert points: %v\n", err)
		os.Exit(1)
	}

	text := fmt.Sprintf("Successfully upserted\nOperation ID: %d\nStatus: %s", upsertResp.OperationId, upsertResp.Status)
	outputResult(*output, text, map[string]any{
		"operationId": upsertResp.OperationId,
		"status":      upsertResp.Status.String(),
		"chunks":      len(chunks),
	})
}

// ---- delete-index ----

func runDeleteIndex(args []string) {
	fs := flag.NewFlagSet("delete-index", flag.ExitOnError)
	envFile := fs.String("env", "", "path to .env file")
	output := fs.String("output", "text", "output format: text or json")
	collection := fs.String("collection", "", "collection name (required)")
	filePath := fs.String("file-path", "", "file path of content to delete (required)")
	fs.Parse(args)

	loadEnv(*envFile)
	checkEnv()

	if *collection == "" || *filePath == "" {
		fmt.Fprintln(os.Stderr, "error: --collection and --file-path are required")
		fs.Usage()
		os.Exit(1)
	}

	ctx := context.Background()
	client := newQdrantClient()

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
										Text: *filePath,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	deleteResp, err := client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: *collection,
		Points:         pointsSelector,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to delete points for filePath %s: %v\n", *filePath, err)
		os.Exit(1)
	}

	text := fmt.Sprintf("Successfully deleted points for filePath: %s\nOperation ID: %d\nStatus: %s", *filePath, deleteResp.OperationId, deleteResp.Status)
	outputResult(*output, text, map[string]any{
		"filePath":    *filePath,
		"operationId": deleteResp.OperationId,
		"status":      deleteResp.Status.String(),
	})
}

// ---- search ----

func runSearch(args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	envFile := fs.String("env", "", "path to .env file")
	output := fs.String("output", "text", "output format: text or json")
	collection := fs.String("collection", "", "collection name (required)")
	query := fs.String("query", "", "search query (required)")
	fs.Parse(args)

	loadEnv(*envFile)
	checkEnv()

	if *collection == "" || *query == "" {
		fmt.Fprintln(os.Stderr, "error: --collection and --query are required")
		fs.Usage()
		os.Exit(1)
	}

	ctx := context.Background()

	resp, err := services.DefaultOpenAIClient().CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input:      []string{*query},
		Model:      openai.LargeEmbedding3,
		Dimensions: 2048,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to generate embeddings for query: %v\n", err)
		os.Exit(1)
	}

	client := newQdrantClient()
	scoreThreshold := float32(0.6)
	searchResult, err := client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: *collection,
		Query:          qdrant.NewQuery(resp.Data[0].Embedding...),
		ScoreThreshold: &scoreThreshold,
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to search in Qdrant: %v\n", err)
		os.Exit(1)
	}

	type searchHit struct {
		Index    int     `json:"index"`
		Score    float32 `json:"score"`
		FilePath string  `json:"filePath"`
		Content  string  `json:"content"`
	}

	var textResult string
	var hits []searchHit
	for i, hit := range searchResult {
		content := hit.Payload["content"].GetStringValue()
		filePath := hit.Payload["filePath"].GetStringValue()
		textResult += fmt.Sprintf("Result %d (Score: %f):\nFilePath: %s\nContent: %s\n\n", i+1, hit.Score, filePath, content)
		hits = append(hits, searchHit{
			Index:    i + 1,
			Score:    hit.Score,
			FilePath: filePath,
			Content:  content,
		})
	}

	if len(hits) == 0 {
		textResult = "No results found."
	}

	outputResult(*output, textResult, map[string]any{"results": hits})
}
