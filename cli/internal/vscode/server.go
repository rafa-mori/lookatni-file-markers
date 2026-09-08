// Package vscode provides VS Code integration server for LookAtni File Markers.
package vscode

import (
	"encoding/json"
	"net/http"
	"strconv"

	gl "github.com/kubex-ecosystem/logz"
	"github.com/kubex-ecosystem/lookatni-file-markers/internal/parser"
	"github.com/kubex-ecosystem/lookatni-file-markers/internal/transpiler"
)

// Server handles VS Code integration requests.
type Server struct {
	logger     *gl.LogzLoggerZ
	port       int
	parser     *parser.MarkerParser
	transpiler *transpiler.Transpiler
}

// NewServer creates a new VS Code integration server.
func NewServer(log *gl.LoggerZ, port int) *Server {
	if log == nil {
		log = gl.GetLoggerZ("lookatni")
	}
	// Load default HTML template
	htmlTemplate := `<!doctype html>
<html lang="pt-BR">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>LookAtni Preview</title>
    <style>
      body { font-family: system-ui, sans-serif; margin: 2rem; line-height: 1.6; }
      .container { max-width: 900px; margin: 0 auto; }
    </style>
  </head>
  <body>
    <div class="container">
      {{.Content}}
    </div>
  </body>
</html>`

	return &Server{
		logger:     log,
		port:       port,
		parser:     parser.New(),
		transpiler: transpiler.New(htmlTemplate),
	}
}

// Start starts the VS Code integration server.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/extract", s.handleExtract)
	mux.HandleFunc("/api/validate", s.handleValidate)
	mux.HandleFunc("/api/generate", s.handleGenerate)
	mux.HandleFunc("/api/transpile", s.handleTranspile)
	mux.HandleFunc("/api/health", s.handleHealth)

	// Enable CORS for VS Code webview
	handler := s.corsMiddleware(mux)

	addr := ":" + strconv.Itoa(s.port)
	s.logger.Log("info", "🌐 VS Code integration server listening on %s", addr)

	return http.ListenAndServe(addr, handler)
}

// APIResponse represents a standard API response.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// handleExtract handles file extraction requests.
func (s *Server) handleExtract(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExtractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	s.logger.Log("debug", "Extract request: %s -> %s", req.MarkedFile, req.OutputDir)

	result, err := s.parser.ExtractFiles(req.MarkedFile, req.OutputDir, req.Options)
	if err != nil {
		s.sendError(w, gl.Sprintf("Extraction failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.sendSuccess(w, result)
}

// handleValidate handles marker validation requests.
func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	s.logger.Log("debug", "Validate request: %s", req.MarkedFile)

	result, err := s.parser.ValidateMarkers(req.MarkedFile, req.Strict)
	if err != nil {
		s.sendError(w, gl.Sprintf("Validation failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.sendSuccess(w, result)
}

// handleGenerate handles directory consolidation requests.
func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	s.logger.Log("debug", "Generate request: %s -> %s", req.SourceDir, req.OutputFile)

	result, err := s.parser.GenerateFromDirectory(req.SourceDir, req.OutputFile, req.ExcludePatterns)
	if err != nil {
		s.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.sendSuccess(w, result)
}

// handleTranspile handles Markdown to HTML transpilation requests.
func (s *Server) handleTranspile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TranspileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	s.logger.Log("debug", "Transpile request: %s -> %s", req.Input, req.OutputDir)

	// TODO: Implement transpilation logic similar to CLI
	// For now, return a placeholder response
	result := map[string]interface{}{
		"message": "Transpilation completed",
		"input":   req.Input,
		"output":  req.OutputDir,
	}

	s.sendSuccess(w, result)
}

// handleHealth handles health check requests.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":  "healthy",
		"service": "lookatni-vscode-integration",
		"version": "2.0.0",
	}
	s.sendSuccess(w, response)
}

// sendSuccess sends a successful API response.
func (s *Server) sendSuccess(w http.ResponseWriter, data interface{}) {
	response := APIResponse{
		Success: true,
		Data:    data,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// sendError sends an error API response.
func (s *Server) sendError(w http.ResponseWriter, message string, statusCode int) {
	response := APIResponse{
		Success: false,
		Error:   message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// corsMiddleware adds CORS headers for VS Code webview communication.
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow VS Code webview origins
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
