# RAG Kit - Model Context Protocol (MCP) Server

The Model Context Protocol (MCP) implementation in RAG Kit enables AI models to interact with vector databases for Retrieval-Augmented Generation (RAG) through a standardized interface.

## Prerequisites

- Go 1.23.2 or higher
- Qdrant vector database
- OpenAI API access

## Installation

### Installing via Go

1. Install the server:

```bash
go install github.com/nguyenvanduocit/rag-kit@latest
```

2. Create a `.env` file with your configuration:

```env
# Required for Qdrant
QDRANT_HOST=     # Required: Qdrant server host
QDRANT_PORT=     # Required: Qdrant server port
QDRANT_API_KEY=  # Required: Qdrant API key

# Required for OpenAI
OPENAI_API_KEY=  # Required: OpenAI API key

# Optional configurations
ENABLE_TOOLS=    # Optional: Comma-separated list of tool groups to enable (empty = all enabled)
PROXY_URL=       # Optional: HTTP/HTTPS proxy URL if needed
```

3. Configure your Claude's config:

```json
{
  "mcpServers": {
    "rag_kit": {
      "command": "rag-kit",
      "args": ["-env", "/path/to/.env"]
    }
  }
}
```

## Enable Tools

The `ENABLE_TOOLS` environment variable is a comma-separated list of tool groups to enable. Available groups are:

- `rag` - RAG (Retrieval-Augmented Generation) tools

Leave it empty to enable all tools.

## Available Tools

### Group: rag

#### RAG_memory_index_content

Index a content into memory, can be inserted or updated

#### RAG_memory_index_file

Index a local file into memory

#### RAG_memory_create_collection

Create a new vector collection in memory

#### RAG_memory_delete_collection

Delete a vector collection in memory

#### RAG_memory_list_collections

List all vector collections in memory

#### RAG_memory_search

Search for memory in a collection based on a query

#### RAG_memory_delete_index_by_filepath

Delete a vector index by filePath

