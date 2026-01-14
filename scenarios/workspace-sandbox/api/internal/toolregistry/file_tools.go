// Package toolregistry provides the tool discovery service for workspace-sandbox.
//
// This file defines the file operation tools (Tier 3) that are exposed
// via the Tool Discovery Protocol.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// FileToolProvider provides the file operation tools.
// These tools enable reading, writing, and managing files in sandboxes.
type FileToolProvider struct{}

// NewFileToolProvider creates a new FileToolProvider.
func NewFileToolProvider() *FileToolProvider {
	return &FileToolProvider{}
}

// Name returns the provider identifier.
func (p *FileToolProvider) Name() string {
	return "workspace-sandbox-files"
}

// Categories returns the tool categories for file operations.
func (p *FileToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:          "file_operations",
			Name:        "File Operations",
			Description: "Tools for reading, writing, and managing files in sandboxes",
			Icon:        "file",
		},
	}
}

// Tools returns the tool definitions for file operations.
func (p *FileToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.listFilesTool(),
		p.readFileTool(),
		p.writeFileTool(),
		p.deleteFileTool(),
		p.createDirectoryTool(),
	}
}

func (p *FileToolProvider) listFilesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_files",
		Description: "List files and directories at a given path within a sandbox. Returns file names, types, sizes, and modification times.",
		Category:    "file_operations",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The sandbox to list files in.",
				},
				"path": {
					Type:        "string",
					Default:     StringValue("."),
					Description: "Path relative to the sandbox root. Defaults to root directory.",
				},
				"recursive": {
					Type:        "boolean",
					Default:     BoolValue(false),
					Description: "If true, recursively list all files in subdirectories.",
				},
				"include_hidden": {
					Type:        "boolean",
					Default:     BoolValue(false),
					Description: "If true, include hidden files (starting with '.').",
				},
				"pattern": {
					Type:        "string",
					Description: "Glob pattern to filter files (e.g., '*.ts', '**/*.go').",
				},
			},
			Required: []string{"sandbox_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"files", "list", "directory"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List files in the src directory",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"path":       "src",
					},
				),
				NewToolExample(
					"Find all TypeScript files recursively",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"recursive":  true,
						"pattern":    "*.ts",
					},
				),
			},
		},
	}
}

func (p *FileToolProvider) readFileTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "read_file",
		Description: "Read the contents of a file in a sandbox. Returns the file content as a string (for text files) or base64-encoded (for binary files).",
		Category:    "file_operations",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The sandbox containing the file.",
				},
				"path": {
					Type:        "string",
					Description: "Path to the file relative to the sandbox root.",
				},
				"encoding": {
					Type:        "string",
					Enum:        []string{"utf-8", "base64", "auto"},
					Default:     StringValue("auto"),
					Description: "Encoding for the file content. 'auto' detects based on file type.",
				},
				"start_line": {
					Type:        "integer",
					Description: "For text files, start reading from this line number (1-indexed).",
				},
				"end_line": {
					Type:        "integer",
					Description: "For text files, stop reading at this line number (inclusive).",
				},
			},
			Required: []string{"sandbox_id", "path"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 120,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"files", "read", "content"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Read a TypeScript file",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"path":       "src/index.ts",
					},
				),
				NewToolExample(
					"Read specific lines from a file",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"path":       "src/large-file.ts",
						"start_line": 100,
						"end_line":   150,
					},
				),
			},
		},
	}
}

func (p *FileToolProvider) writeFileTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "write_file",
		Description: "Write content to a file in a sandbox. Creates the file if it doesn't exist, or overwrites if it does. Can optionally create parent directories.",
		Category:    "file_operations",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The sandbox to write the file in.",
				},
				"path": {
					Type:        "string",
					Description: "Path to the file relative to the sandbox root.",
				},
				"content": {
					Type:        "string",
					Description: "Content to write to the file.",
				},
				"encoding": {
					Type:        "string",
					Enum:        []string{"utf-8", "base64"},
					Default:     StringValue("utf-8"),
					Description: "Encoding of the provided content. Use 'base64' for binary files.",
				},
				"create_dirs": {
					Type:        "boolean",
					Default:     BoolValue(true),
					Description: "If true, create parent directories if they don't exist.",
				},
				"mode": {
					Type:        "string",
					Default:     StringValue("0644"),
					Description: "File permissions in octal format (e.g., '0755' for executable).",
				},
			},
			Required: []string{"sandbox_id", "path", "content"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         false,
			ModifiesState:      true,
			Tags:               []string{"files", "write", "create"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Write a new TypeScript file",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"path":       "src/utils/helper.ts",
						"content":    "export function helper() { return 'hello'; }",
					},
				),
				NewToolExample(
					"Write an executable script",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"path":       "scripts/build.sh",
						"content":    "#!/bin/bash\nnpm run build",
						"mode":       "0755",
					},
				),
			},
		},
	}
}

func (p *FileToolProvider) deleteFileTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "delete_file",
		Description: "Delete a file or directory from a sandbox. For directories, use 'recursive' to delete all contents.",
		Category:    "file_operations",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The sandbox containing the file/directory.",
				},
				"path": {
					Type:        "string",
					Description: "Path to the file or directory relative to the sandbox root.",
				},
				"recursive": {
					Type:        "boolean",
					Default:     BoolValue(false),
					Description: "If true, recursively delete directory contents. Required for non-empty directories.",
				},
			},
			Required: []string{"sandbox_id", "path"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      true,
			Tags:               []string{"files", "delete", "remove"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Delete a single file",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"path":       "src/old-file.ts",
					},
				),
				NewToolExample(
					"Delete a directory and its contents",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"path":       "src/deprecated",
						"recursive":  true,
					},
				),
			},
		},
	}
}

func (p *FileToolProvider) createDirectoryTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "create_directory",
		Description: "Create a new directory in a sandbox. Can optionally create all parent directories.",
		Category:    "file_operations",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The sandbox to create the directory in.",
				},
				"path": {
					Type:        "string",
					Description: "Path to the directory relative to the sandbox root.",
				},
				"parents": {
					Type:        "boolean",
					Default:     BoolValue(true),
					Description: "If true, create all parent directories if they don't exist (like mkdir -p).",
				},
				"mode": {
					Type:        "string",
					Default:     StringValue("0755"),
					Description: "Directory permissions in octal format.",
				},
			},
			Required: []string{"sandbox_id", "path"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     15,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      true,
			Tags:               []string{"files", "directory", "create"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Create a nested directory structure",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"path":       "src/components/common",
					},
				),
			},
		},
	}
}
