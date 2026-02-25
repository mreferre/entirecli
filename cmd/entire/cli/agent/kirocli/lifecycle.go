package kirocli

import "github.com/entireio/cli/cmd/entire/cli/agent"

// Compile-time interface assertions for optional interfaces.
// These verify that KiroCLIAgent properly implements:
// - TranscriptAnalyzer: format-specific transcript parsing
// - TokenCalculator: token usage calculation from transcripts
var (
	_ agent.TranscriptAnalyzer = (*KiroCLIAgent)(nil)
	_ agent.TokenCalculator    = (*KiroCLIAgent)(nil)
)
