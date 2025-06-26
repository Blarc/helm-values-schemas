package main

import (
	"encoding/json"
	"fmt"
	"github.com/losisin/helm-values-schema-json/pkg"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type SchemaCache struct {
	mu    sync.RWMutex
	cache map[string][]byte
}

func NewSchemaCache() *SchemaCache {
	return &SchemaCache{
		cache: make(map[string][]byte),
	}
}

func (c *SchemaCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	schema, ok := c.cache[key]
	return schema, ok
}

func (c *SchemaCache) Set(key string, schema []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = schema
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func downloadValuesFile(valuesUrl string) ([]byte, error) {
	valuesUrl = fmt.Sprintf("%s%s", "https://raw.githubusercontent.com", valuesUrl)
	log.Printf("Downloading values file from: %s", valuesUrl)

	// Create HTTP request
	req, err := http.NewRequest("GET", valuesUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", "helm-values-schema-generator/1.0")
	req.Header.Set("Accept", "text/plain, application/x-yaml, */*")

	// Make the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download file: HTTP %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("Successfully downloaded %d bytes from %s", len(body), valuesUrl)
	return body, nil
}

func generateJSONSchema(valuesUrl string, valuesContent []byte) ([]byte, error) {
	tmpFile, err := os.CreateTemp("", "values-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	tmpSchemaFile, err := os.CreateTemp("", "values-*.schema.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp schema file: %w", err)
	}
	defer os.Remove(tmpSchemaFile.Name())
	defer tmpSchemaFile.Close()

	// Write value content to a temporary file
	if _, err := tmpFile.Write(valuesContent); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}

	// Close the file so it can be read by the schema generator
	tmpFile.Close()

	trueBoolFlag := pkg.BoolFlag{}
	trueBoolFlag.Set("true")

	// Generate schema using the helm-values-schema-json library
	config := &pkg.Config{
		Input:      []string{tmpFile.Name()},
		OutputPath: tmpSchemaFile.Name(),
		Draft:      2020,
		Indent:     4,
		SchemaRoot: pkg.SchemaRoot{
			ID:                   "https://example.com/schema",
			Title:                "Helm Values Schema",
			Description:          fmt.Sprintf("Generated Helm Values Schema for %s", valuesUrl),
			AdditionalProperties: trueBoolFlag,
		},
	}

	err = pkg.GenerateJsonSchema(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema: %w", err)
	}

	// Close the schema file before reading it
	tmpSchemaFile.Close()

	// Read the generated schema file content
	schemaContent, err := os.ReadFile(tmpSchemaFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read generated schema file: %w", err)
	}

	return schemaContent, nil
}

func main() {
	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize cache
	cache := NewSchemaCache()

	// Define routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleSchemaRequest(w, r, cache)
	})

	// Start server
	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// Handle schema generation requests
func handleSchemaRequest(w http.ResponseWriter, r *http.Request, cache *SchemaCache) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		sendErrorResponse(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	// Parse query parameters
	valuesUrl := r.URL.Path
	log.Println(valuesUrl)

	// Validate parameters
	if valuesUrl == "/" {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Hello World")
		return
	}

	// Create a cache key
	cacheKey := valuesUrl

	// Check if schema is in cache
	if schema, found := cache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/schema+json")
		w.WriteHeader(http.StatusOK)
		w.Write(schema)
		return
	}

	// Download the value file
	valuesContent, err := downloadValuesFile(valuesUrl)
	if err != nil {
		log.Printf("Error downloading values file: %v", err)
		sendErrorResponse(w, http.StatusBadRequest, "download_failed",
			fmt.Sprintf("Failed to download values file: %v", err))
		return
	}

	// TODO: Process valuesContent to generate JSON schema
	// For now, we'll create a placeholder schema
	schema, err := generateJSONSchema(valuesUrl, valuesContent)

	// Cache the result
	cache.Set(cacheKey, schema)

	// Return the schema
	w.Header().Set("Content-Type", "application/schema+json")
	w.WriteHeader(http.StatusOK)
	w.Write(schema)
}

// Send error response in JSON format
func sendErrorResponse(w http.ResponseWriter, statusCode int, errorType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error:   errorType,
		Message: message,
	}

	json.NewEncoder(w).Encode(response)
}
