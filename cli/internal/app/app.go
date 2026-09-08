// Package app provides the main CLI application logic for LookAtni File Markers.
package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	// "github.com/kubex-ecosystem/logz"
	gl "github.com/kubex-ecosystem/logz"
	"github.com/kubex-ecosystem/lookatni-file-markers/internal/adaptive"
	"github.com/kubex-ecosystem/lookatni-file-markers/internal/integration"
	"github.com/kubex-ecosystem/lookatni-file-markers/internal/metadata"
	"github.com/kubex-ecosystem/lookatni-file-markers/internal/module/kbx"
	"github.com/kubex-ecosystem/lookatni-file-markers/internal/parser"
	"github.com/kubex-ecosystem/lookatni-file-markers/internal/transpiler"
)

// App represents the main CLI application.
type App struct {
	log               *gl.LogzLoggerZ // Is already a interface, so, a pointer...
	parser            *parser.MarkerParser
	transpiler        *transpiler.Transpiler
	gromptIntegration *integration.GromptIntegration
}

func init() {
	// Initialize Grompt integration
	gromptIntegration := integration.NewGromptIntegration()
	providers := gromptIntegration.GetAvailableProviders()
	if len(providers) > 0 {
		gl.Log("info", fmt.Sprintf("Grompt integration initialized with %d providers", len(providers)))
	}
}

// New creates a new App instance.
func New(log *gl.LogzLoggerZ) *App {
	if log == nil {
		log = gl.GetLoggerZ("lookatni")
	}

	// Load HTML template
	// htmlTemplate, err := templatesFS.ReadFile("templates/base.html")
	// if err != nil {
	// 	log.Log("error", "Failed to load HTML template: %v", err)
	// 	// Fallback to embedded template
	// 	htmlTemplate = []byte(fallbackHTMLTemplate)
	// }

	return &App{
		log:               log,
		parser:            parser.New(),
		transpiler:        transpiler.New( /* string(htmlTemplate) */ ""),
		gromptIntegration: integration.NewGromptIntegration(),
	}
}

// Run executes the CLI application with the given arguments.
func (a *App) Run(args []string) error {
	if len(args) == 0 {
		return a.showHelp()
	}
	command := args[0]
	switch command {
	case "extract":
		return a.extractCommand(args[1:])
	case "validate":
		return a.validateCommand(args[1:])
	case "generate":
		return a.generateCommand(args[1:])
	case "transpile":
		return a.transpileCommand(args[1:]...)
	case "refactor":
		return a.refactorCommand(args[1:])
	case "help":
		return a.showHelp()
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// extractCommand handles file extraction from marked files.
func (a *App) extractCommand(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: extract <marked-file> <output-dir> [--overwrite] [--create-dirs] [--dry-run]")
	}

	markedFile := args[0]
	outputDir := args[1]

	options := kbx.ExtractOptions{
		Overwrite:  false,
		CreateDirs: true,
		DryRun:     false,
	}

	// Parse flags
	for _, arg := range args[2:] {
		switch arg {
		case "--overwrite":
			options.Overwrite = true
		case "--create-dirs":
			options.CreateDirs = true
		case "--dry-run":
			options.DryRun = true
		}
	}

	a.log.Log("info", fmt.Sprintf("Extracting files from %s to %s", markedFile, outputDir))

	result, err := a.parser.ExtractFiles(markedFile, outputDir, options)
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	if len(result.Errors) > 0 {
		a.log.Log("warn", "Extraction completed with errors:")
		for _, errMsg := range result.Errors {
			a.log.Log("warn", "   %s", errMsg)
		}
	}

	if options.DryRun {
		a.log.Log("info", "[DRY RUN] Would extract %d files", len(result.ExtractedFiles))
	} else {
		a.log.Log("success", fmt.Sprintf("Successfully extracted %d files", len(result.ExtractedFiles)))
	}

	for _, file := range result.ExtractedFiles {
		if options.DryRun {
			a.log.Log("debug", fmt.Sprintf("   [DRY RUN] %s", file))
		} else {
			a.log.Log("debug", fmt.Sprintf("   ✓ %s", file))
		}
	}

	return nil
}

// validateCommand handles marker validation.
func (a *App) validateCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: validate <marked-file> [--strict]")
	}

	markedFile := args[0]
	strict := false
	for _, arg := range args[1:] {
		if arg == "--strict" {
			strict = true
		}
	}

	a.log.Log("info", fmt.Sprintf("Validating markers in %s (strict=%v)", markedFile, strict))

	result, err := a.parser.ValidateMarkers(markedFile, strict)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if result.IsValid {
		a.log.Log("success", "All markers are valid!")
	} else {
		a.log.Log("warn", "Validation issues found:")
	}

	if len(result.Errors) > 0 {
		a.log.Log("warn", "Errors:")
		for _, errMsg := range result.Errors {
			a.log.Log("warn", "   Line %d: %s (%s)", errMsg.Line, errMsg.Message, errMsg.Severity)
		}
	}

	if len(result.DuplicateFilenames) > 0 {
		a.log.Log("warn", "Duplicate filenames:")
		for _, dup := range result.DuplicateFilenames {
			a.log.Log("warn", "   %s", dup)
		}
	}

	if len(result.InvalidFilenames) > 0 {
		a.log.Log("warn", "Invalid filenames:")
		for _, invalid := range result.InvalidFilenames {
			a.log.Log("warn", "   %s", invalid)
		}
	}

	stats := result.Statistics
	a.log.Log("info", "📊 Statistics:")
	a.log.Log("info", "   Total markers: %d", stats.TotalMarkers)
	a.log.Log("info", "   Empty markers: %d", stats.EmptyMarkers)

	return nil
}

// transpileCommand handles Markdown to HTML transpilation.
func (a *App) transpileCommand(args ...string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: transpile <input-dir|file> <output-dir> [--with-prompts]")
	}

	input := args[0]
	outputDir := args[1]

	// Check for prompt processing flag
	withPrompts := false
	for _, arg := range args[2:] {
		if arg == "--with-prompts" {
			withPrompts = true
			break
		}
	}

	a.log.Log("info", fmt.Sprintf("Transpiling from %s to %s (prompts: %v)", input, outputDir, withPrompts))

	stat, err := os.Stat(input)
	if err != nil {
		return fmt.Errorf("input not found: %w", err)
	}

	var files []transpiler.FileInfo
	var totalSize int64

	if stat.IsDir() {
		// Process directory
		entries, err := os.ReadDir(input)
		if err != nil {
			return fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			if !strings.HasPrefix(entry.Name(), "interview_") {
				continue
			}

			filePath := filepath.Join(input, entry.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				a.log.Log("warn", fmt.Sprintf("Failed to read %s: %v", filePath, err))
				continue
			}

			// Process with Grompt if enabled
			if withPrompts && a.gromptIntegration != nil {
				processedContent, err := a.gromptIntegration.ProcessMarkdownWithPrompts(string(content))
				if err != nil {
					a.log.Log("warn", fmt.Sprintf("Grompt processing failed for %s: %v", entry.Name(), err))
				} else {
					content = []byte(processedContent)
					a.log.Log("debug", fmt.Sprintf("Enhanced %s with AI processing", entry.Name()))
				}
			}

			fileInfo, err := a.transpiler.ConvertMarkdownToHTML(entry.Name(), content, outputDir)
			if err != nil {
				a.log.Log("warn", "Failed to transpile %s: %v", entry.Name(), err)
				continue
			}

			files = append(files, *fileInfo)

			// Calculate size
			if stat, err := os.Stat(filepath.Join(outputDir, fileInfo.FileName)); err == nil {
				totalSize += stat.Size()
			}

			a.log.Log("debug", fmt.Sprintf("  ✓ %s -> %s", entry.Name(), fileInfo.FileName))
		}
	} else {
		// Process single file
		if !strings.HasSuffix(input, ".md") {
			return fmt.Errorf("input must be a .md file")
		}

		content, err := os.ReadFile(input)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		filename := filepath.Base(input)
		fileInfo, err := a.transpiler.ConvertMarkdownToHTML(filename, content, outputDir)
		if err != nil {
			return fmt.Errorf("transpilation failed: %w", err)
		}

		files = append(files, *fileInfo)
		if stat, err := os.Stat(filepath.Join(outputDir, fileInfo.FileName)); err == nil {
			totalSize = stat.Size()
		}
	}

	// Generate index
	if len(files) > 0 {
		if err := a.transpiler.GenerateIndex(files, totalSize, outputDir); err != nil {
			return fmt.Errorf("failed to generate index: %w", err)
		}
		a.log.Log("success", fmt.Sprintf("Generated index with %d files", len(files)))
	}

	a.log.Log("success", fmt.Sprintf("Transpilation completed: %d files processed", len(files)))
	return nil
}

// generateCommand handles project consolidation (directory -> marked file).
func (a *App) generateCommand(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: generate <source-dir> <output-file> [--exclude patterns]")
	}

	sourceDir := args[0]
	outputFile := args[1]

	// Parse flags from args
	var excludePatterns []string
	var markerPreset, markerStart, markerEnd, markerPattern string
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--exclude":
			if i+1 < len(args) {
				excludePatterns = append(excludePatterns, args[i+1])
				i++
			}
		case "--marker-preset":
			if i+1 < len(args) {
				markerPreset = args[i+1]
				i++
			}
		case "--marker-start":
			if i+1 < len(args) {
				markerStart = args[i+1]
				i++
			}
		case "--marker-end":
			if i+1 < len(args) {
				markerEnd = args[i+1]
				i++
			}
		case "--marker-pattern":
			if i+1 < len(args) {
				markerPattern = args[i+1]
				i++
			}
		}
	}

	// Default exclude patterns
	if len(excludePatterns) == 0 {
		excludePatterns = []string{
			"*.git*", "node_modules", "dist", "build", "*.log", "*.tmp",
		}
	}

	a.log.Log("info", fmt.Sprintf("Generating marked file from %s to %s", sourceDir, outputFile))

	// If custom marker parameters provided, use adaptive generator
	if markerPreset != "" || markerStart != "" || markerEnd != "" || markerPattern != "" {
		ap := adaptive.New()
		// Compose config
		var cfg metadata.MarkerConfig
		if markerPreset != "" {
			presets := metadata.GetPresetConfigs()
			if p, ok := presets[markerPreset]; ok {
				cfg = p.Config
			} else {
				cfg = metadata.GetDefaultConfig()
			}
		} else {
			cfg = metadata.GetDefaultConfig()
		}
		if markerPattern != "" {
			cfg.Pattern = markerPattern
		}
		if markerStart != "" {
			cfg.Start = markerStart
		}
		if markerEnd != "" {
			cfg.End = markerEnd
		}

		res, err := ap.GenerateFromDirectory(sourceDir, outputFile, excludePatterns, &cfg)
		if err != nil {
			return fmt.Errorf("generation failed (adaptive): %w", err)
		}
		// Map to log summary
		if len(res.Errors) > 0 {
			a.log.Log("warn", "Generation completed with warnings:")
			for _, e := range res.Errors {
				a.log.Log("warn", "   %s", e)
			}
		}
		a.log.Log("success", "Successfully generated marked file:")
		a.log.Log("success", fmt.Sprintf("   📁 %d files processed", res.TotalFiles))
		a.log.Log("success", fmt.Sprintf("   📊 %d bytes written", res.TotalBytes))
		a.log.Log("success", fmt.Sprintf("   📄 Output: %s", outputFile))
		return nil
	}

	result, err := a.parser.GenerateFromDirectory(sourceDir, outputFile, excludePatterns)
	if err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	if len(result.Errors) > 0 {
		a.log.Log("warn", "Generation completed with warnings:")
		for _, errMsg := range result.Errors {
			a.log.Log("warn", fmt.Sprintf("   %s", errMsg))
		}
	}

	a.log.Log("success", "Successfully generated marked file:")
	a.log.Log("success", fmt.Sprintf("   📁 %d files processed", result.TotalFiles))
	a.log.Log("success", fmt.Sprintf("   📊 %d bytes written", result.TotalBytes))
	a.log.Log("success", fmt.Sprintf("   📄 Output: %s", outputFile))

	return nil
}

// refactorCommand handles AI-powered code refactoring using Grompt integration.
func (a *App) refactorCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: refactor <artifact-file> [--rules <file>] [--provider <name>] [--output <dir>] [--dry-run] [--interactive]")
	}

	artifactFile := args[0]

	// Default options
	rulesFile := "docs/prompt/my-rules.md"
	provider := "gemini" // Default to gemini since we have the key
	outputDir := ""
	dryRun := false
	interactive := false

	// Parse flags
	for i, arg := range args[1:] {
		switch arg {
		case "--rules":
			if i+2 < len(args) {
				rulesFile = args[i+2]
			}
		case "--provider":
			if i+2 < len(args) {
				provider = args[i+2]
			}
		case "--output":
			if i+2 < len(args) {
				outputDir = args[i+2]
			}
		case "--dry-run":
			dryRun = true
		case "--interactive":
			interactive = true
		}
	}

	a.log.Log("info", "🤖 Starting AI-powered refactoring...")
	a.log.Log("info", fmt.Sprintf("📄 Artifact: %s", artifactFile))
	a.log.Log("info", fmt.Sprintf("📋 Rules: %s", rulesFile))
	a.log.Log("info", fmt.Sprintf("🧠 Provider: %s", provider))

	// Read artifact content
	artifactContent, err := os.ReadFile(artifactFile)
	if err != nil {
		return fmt.Errorf("failed to read artifact file: %w", err)
	}

	// Read rules content
	rulesContent, err := os.ReadFile(rulesFile)
	if err != nil {
		a.log.Log("warn", fmt.Sprintf("Failed to read rules file %s, using default rules", rulesFile))
		rulesContent = []byte("Apply Go best practices and improve code quality")
	}

	if a.gromptIntegration == nil {
		return fmt.Errorf("grompt integration not initialized")
	}

	// Perform refactoring using Grompt
	a.log.Log("info", "🔄 Processing with AI provider...")
	result, err := a.gromptIntegration.RefactorArtifact(
		string(artifactContent),
		string(rulesContent),
		provider,
	)
	if err != nil {
		return fmt.Errorf("refactoring failed: %w", err)
	}

	a.log.Log("info", "✅ Refactoring completed - ID: %s", result.ProcessingID)

	if dryRun {
		// Just show suggestions
		a.log.Log("info", "🔍 DRY RUN - Suggestions:")
		for i, suggestion := range result.Suggestions {
			a.log.Log("info", "  %d. %s", i+1, suggestion)
		}
		return nil
	}

	if interactive {
		// TODO: Implement interactive mode
		a.log.Log("info", "🤝 Interactive mode not yet implemented, proceeding with automatic refactoring")
	}

	// Save refactored content
	outputFile := artifactFile
	if outputDir != "" {
		outputFile = filepath.Join(outputDir, filepath.Base(artifactFile))
	}

	// Add .refactored suffix if not overwriting
	if outputFile == artifactFile {
		ext := filepath.Ext(outputFile)
		base := strings.TrimSuffix(outputFile, ext)
		outputFile = base + ".refactored" + ext
	}

	err = os.WriteFile(outputFile, []byte(result.RefactoredContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write refactored file: %w", err)
	}

	a.log.Log("info", fmt.Sprintf("✅ Refactored artifact saved to: %s", outputFile))
	a.log.Log("info", fmt.Sprintf("📊 Applied %d suggestions", len(result.Suggestions)))

	return nil
}

// showHelp displays help information.
func (a *App) showHelp() error {
	help := `LookAtni File Markers v2.0 - Advanced file organization with Go power

Usage:
  lookatni <command> [options]

Commands:
  extract <marked-file> <output-dir> [flags]  Extract files FROM marked content
  validate <marked-file>                      Validate markers in consolidated file
  generate <source-dir> <output-file> [flags] Consolidate directory INTO marked file
  transpile <input> <output-dir> [flags]      Convert Markdown to HTML with AI
  help                                        Show this help

Global Flags:
  --list-presets                              List available marker presets
  --version                                   Show version information
  --vscode                                    Run in VS Code integration mode
  --port <num>                                Port for VS Code server (default: 8080)
  -v                                          Enable verbose logging

Extract Flags:
  --overwrite     Overwrite existing files
  --create-dirs   Create directories as needed
  --dry-run       Show what would be done without doing it

Generate Flags:
  --exclude <pattern>  Exclude files matching pattern (can be used multiple times)

Transpile Flags:
  --with-prompts  Enable AI-powered content enhancement via Grompt integration

🎨 Custom Markers (PREVIEW):
  The new adaptive marker system supports multiple formats:
  • HTML Comments: <!-- FILE: filename -->
  • Markdown Invisible: [//]: # (FILE: filename)
  • Code Comments: // === FILE: filename ===
  • Visual Separators: FILE: filename
  • Classic (ASCII 28): Invisible markers (default)

Examples:
  # Basic workflow
  lookatni generate ./my-project project.marked --exclude "*.log" --exclude "node_modules"
  lookatni extract project.marked ./output --overwrite --create-dirs
  lookatni validate project.marked

  # New features with AI integration
  lookatni --list-presets                     # Show available marker formats
  lookatni transpile ./interviews ./output    # Convert Markdown to HTML
  lookatni transpile ./docs ./output --with-prompts  # AI-enhanced transpilation

  # VS Code integration
  lookatni --vscode --port 8080               # Start integration server

For more info: https://github.com/kubex-ecosystem/lookatni-file-markers
`
	fmt.Print(help)
	return nil
}

// fallbackHTMLTemplate is used when the embedded template fails to load.
const fallbackHTMLTemplate = `<!doctype html>
<html lang="pt-BR">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Interview Prep</title>
    <style>
      body { font-family: system-ui, -apple-system, Segoe UI, Roboto, sans-serif; margin: 2rem; line-height: 1.6; }
      pre, code { background: #f6f8fa; padding: .2rem .4rem; border-radius: 4px; }
      a { color: #0b69c7; text-decoration: none; }
      a:hover { text-decoration: underline; }
      h1, h2, h3 { line-height: 1.25; }
      .container { max-width: 900px; margin: 0 auto; }
      kx-prompt-block { display: block; border: 1px solid #e5e7eb; border-radius: 8px; padding: 12px; margin: 16px 0; }
      .kx-field { margin: 8px 0; }
      .kx-controls { display: flex; gap: 8px; margin-top: 8px; }
      .kx-controls button { background: #0b69c7; color: white; border: none; border-radius: 6px; padding: 6px 10px; cursor: pointer; }
      .kx-controls button.secondary { background: #6b7280; }
      .kx-controls button:hover { opacity: .95; }
    </style>
  </head>
  <body>
    <div class="container">
      {{.Content}}
    </div>
    <script>
      // Add interactive functionality for prompt blocks
      document.addEventListener('DOMContentLoaded', function() {
        const promptBlocks = document.querySelectorAll('kx-prompt-block');
        promptBlocks.forEach(block => {
          const runButton = block.querySelector('button.run');
          if (runButton) {
            runButton.addEventListener('click', function() {
              console.log('Run with MCP clicked');
              // TODO: Implement MCP integration
            });
          }
        });
      });
    </script>
  </body>
</html>`
