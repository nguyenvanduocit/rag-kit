package tools

import (
	"context"
	"fmt"

	"github.com/nguyenvanduocit/rag-kit/services"
	"github.com/pkoukk/tiktoken-go"
	"github.com/sashabaranov/go-openai"
)

// SplitIntoChunks splits content into consumable chunks
func SplitIntoChunks(content string, filePath string) ([]string, error) {
	const (
		maxTokensPerChunk = 512
		overlapTokens     = 50
		model             = "text-embedding-3-large"
	)

	encoding, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return nil, fmt.Errorf("failed to get encoding: %v", err)
	}

	tokens := encoding.Encode(content, nil, nil)

	var chunks []string
	var currentChunk []int

	// First pass: collect all chunks without context
	var rawChunks []string
	for i := 0; i < len(tokens); i++ {
		currentChunk = append(currentChunk, tokens[i])

		if len(currentChunk) >= maxTokensPerChunk {
			chunkText := encoding.Decode(currentChunk)
			rawChunks = append(rawChunks, chunkText)

			if len(currentChunk) > overlapTokens {
				currentChunk = currentChunk[len(currentChunk)-overlapTokens:]
			} else {
				currentChunk = []int{}
			}
		}
	}

	// Handle remaining tokens
	if len(currentChunk) > 0 {
		chunkText := encoding.Decode(currentChunk)
		rawChunks = append(rawChunks, chunkText)
	}

	// If there's only one chunk, return it without context
	if len(rawChunks) == 1 {
		return rawChunks, nil
	}

	// If there are multiple chunks, add context to each
	for _, chunkText := range rawChunks {
		contextualizedChunk, err := GenerateContext(content, chunkText)
		if err != nil {
			return nil, fmt.Errorf("failed to generate context: %v", err)
		}
		chunks = append(chunks, contextualizedChunk)
	}

	return chunks, nil
}

// GenerateContext generates context for a chunk
func GenerateContext(fullText, chunkText string) (string, error) {
	prompt := fmt.Sprintf(`
<document>%s</document>
Here is the chunk we want to situate within the whole document:
<chunk>%s</chunk>
Please give a short succinct context to situate this chunk within the overall document for the purposes of improving search retrieval of the chunk. Answer only with the succinct context and nothing else.
	`, fullText, chunkText)

	resp, err := services.DefaultOpenAIClient().CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4oLatest,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to generate context: %v", err)
	}

	context := resp.Choices[0].Message.Content
	return fmt.Sprintf("Context: \n%s;\n\nChunk: \n%s", context, chunkText), nil
} 