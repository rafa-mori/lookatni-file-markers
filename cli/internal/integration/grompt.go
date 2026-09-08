// Package integration demonstrates Grompt integration within LookAtni
package integration

import (
	"fmt"

	"github.com/kubex-ecosystem/grompt"
	gl "github.com/kubex-ecosystem/logz"
)

// GromptIntegration provides prompt engineering capabilities for LookAtni
type GromptIntegration struct {
	engine grompt.PromptEngine
	config grompt.Config
}

// NewGromptIntegration creates a new Grompt integration instance
func NewGromptIntegration() *GromptIntegration {
	config := grompt.DefaultConfig("")
	engine := grompt.NewPromptEngine(config)

	return &GromptIntegration{
		engine: engine,
		// config: config,
	}
}

// ProcessMarkdownWithPrompts processes markdown content with prompt blocks
func (g *GromptIntegration) ProcessMarkdownWithPrompts(content string) (string, error) {
	if g.engine == nil {
		return content, fmt.Errorf("grompt engine not initialized")
	}

	// Example: Process a simple prompt template
	variables := map[string]interface{}{
		"content": content,
		"task":    "markdown_processing",
	}

	template := "Process this markdown content: {{.content}} for task: {{.task}}"

	result, err := g.engine.ProcessPrompt(template, variables)
	if err != nil {
		gl.Log("warn", fmt.Sprintf("Failed to process prompt: %v", err))
		return content, nil // Return original content on error
	}

	gl.Log("info", fmt.Sprintf("✅ Processed markdown with Grompt - ID: %s", result.ID))
	return result.Response, nil
}

// GetAvailableProviders returns the list of available AI providers
func (g *GromptIntegration) GetAvailableProviders() []grompt.Provider {
	if g.engine == nil {
		return nil
	}
	return g.engine.GetProviders()
}

// GetProcessingHistory returns the history of processed prompts
func (g *GromptIntegration) GetProcessingHistory() []grompt.Result {
	if g.engine == nil {
		return nil
	}
	return g.engine.GetHistory()
}

// BatchProcessPrompts processes multiple prompts concurrently
func (g *GromptIntegration) BatchProcessPrompts(prompts []string, commonVars map[string]interface{}) ([]grompt.Result, error) {
	if g.engine == nil {
		return nil, fmt.Errorf("grompt engine not initialized")
	}

	return g.engine.BatchProcess(prompts, commonVars)
}

// SaveInteraction manually saves a prompt/response pair to history
func (g *GromptIntegration) SaveInteraction(prompt, response string) error {
	if g.engine == nil {
		return fmt.Errorf("grompt engine not initialized")
	}

	return g.engine.SaveToHistory(prompt, response)
}

// RefactorArtifact processes a LookAtni artifact with AI-powered refactoring
func (g *GromptIntegration) RefactorArtifact(artifactContent, rulesContent, provider string) (*RefactorResult, error) {
	if g.engine == nil {
		return nil, fmt.Errorf("grompt engine not initialized")
	}

	// Create professional refactoring prompt
	prompt := g.buildRefactorPrompt(artifactContent, rulesContent)

	// Process with specified provider or use the best available
	result, err := g.engine.ProcessPrompt(prompt, map[string]interface{}{
		"provider":         provider,
		"artifact_content": artifactContent,
		"rules_content":    rulesContent,
		"task":             "code_refactoring",
	})

	if err != nil {
		gl.Log("error", fmt.Sprintf("Failed to process refactor prompt: %v", err))
		return nil, fmt.Errorf("failed to process refactor prompt: %w", err)
	}

	gl.Log("info", fmt.Sprintf("✅ Refactored artifact with Grompt - ID: %s", result.ID))

	return &RefactorResult{
		OriginalContent:   artifactContent,
		RefactoredContent: result.Response,
		AppliedRules:      rulesContent,
		Provider:          provider,
		ProcessingID:      result.ID,
		Suggestions:       g.extractSuggestions(result.Response),
	}, nil
}

// AnalyzeCodeQuality analyzes code quality and suggests improvements
func (g *GromptIntegration) AnalyzeCodeQuality(content string) (*QualityAnalysis, error) {
	if g.engine == nil {
		return nil, fmt.Errorf("grompt engine not initialized")
	}

	prompt := g.buildQualityAnalysisPrompt(content)

	result, err := g.engine.ProcessPrompt(prompt, map[string]interface{}{
		"content": content,
		"task":    "quality_analysis",
	})

	if err != nil {
		return nil, fmt.Errorf("failed to analyze code quality: %w", err)
	}

	return &QualityAnalysis{
		OverallScore: g.extractQualityScore(result.Response),
		Issues:       g.extractIssues(result.Response),
		Suggestions:  g.extractSuggestions(result.Response),
		ProcessingID: result.ID,
	}, nil
}

// RefactorResult contains the result of artifact refactoring
type RefactorResult struct {
	OriginalContent   string   `json:"original_content"`
	RefactoredContent string   `json:"refactored_content"`
	AppliedRules      string   `json:"applied_rules"`
	Provider          string   `json:"provider"`
	ProcessingID      string   `json:"processing_id"`
	Suggestions       []string `json:"suggestions"`
}

// QualityAnalysis contains code quality analysis results
type QualityAnalysis struct {
	OverallScore int      `json:"overall_score"`
	Issues       []string `json:"issues"`
	Suggestions  []string `json:"suggestions"`
	ProcessingID string   `json:"processing_id"`
}

// buildRefactorPrompt creates a professional prompt for code refactoring
func (g *GromptIntegration) buildRefactorPrompt(artifactContent, rulesContent string) string {
	return fmt.Sprintf(`# LookAtni Code Refactoring Assistant

You are an expert code refactoring assistant working with LookAtni artifacts. Your task is to analyze the provided code artifact and apply refactoring improvements based on the specified rules.

## Refactoring Rules

%s

## Code Artifact to Refactor

%s

## Instructions

1. **Analyze** the code structure, patterns, and potential improvements
2. **Apply** the refactoring rules systematically
3. **Preserve** the original functionality while improving:
   - Code quality and readability
   - Performance and efficiency
   - Error handling and robustness
   - Documentation and comments
4. **Maintain** the LookAtni artifact format (file markers and structure)
5. **Provide** clear comments explaining major refactoring decisions

## Output Format

Return the refactored artifact with:
- Improved code following the specified rules
- Preserved LookAtni file markers and structure
- Added comments explaining significant changes
- Maintained original functionality

## Refactored Artifact

`, rulesContent, artifactContent)
}

// buildQualityAnalysisPrompt creates a prompt for code quality analysis
func (g *GromptIntegration) buildQualityAnalysisPrompt(content string) string {
	return fmt.Sprintf(`# Code Quality Analysis

Analyze the following code and provide a comprehensive quality assessment:

## Code to Analyze

%s

## Analysis Requirements

1. **Overall Quality Score** (1-10)
2. **Identified Issues** (performance, security, maintainability)
3. **Improvement Suggestions** (specific, actionable recommendations)
4. **Best Practices Compliance** (language-specific standards)

Provide a structured analysis with clear, actionable feedback.

`, content)
}

// extractQualityScore extracts quality score from AI response
func (g *GromptIntegration) extractQualityScore(response string) int {
	// TODO: Implement smart extraction of quality score from response
	// For now, return a placeholder score
	return 7
}

// extractIssues extracts identified issues from AI response
func (g *GromptIntegration) extractIssues(response string) []string {
	// TODO: Implement smart extraction of issues from response
	// For now, return placeholder issues
	return []string{"Issues analysis pending implementation"}
}

// extractSuggestions extracts suggestions from AI response
func (g *GromptIntegration) extractSuggestions(response string) []string {
	// TODO: Implement smart extraction of suggestions from response
	// For now, return placeholder suggestions
	return []string{"Suggestions extraction pending implementation"}
}
