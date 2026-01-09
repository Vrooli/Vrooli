package docs

// Package docs implements the Docs validation phase. It scans Markdown files for
// syntax issues, validates mermaid diagrams, checks links (local and external),
// and guards against absolute filesystem paths so scenarios stay portable.
