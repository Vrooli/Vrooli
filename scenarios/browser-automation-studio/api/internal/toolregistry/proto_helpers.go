// Package toolregistry provides helper functions for building tool definitions.
package toolregistry

import (
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// StringParam creates a string parameter schema.
func StringParam(description string) *toolspb.ParameterSchema {
	return &toolspb.ParameterSchema{
		Type:        "string",
		Description: description,
	}
}

// StringParamWithFormat creates a string parameter schema with format.
func StringParamWithFormat(description, format string) *toolspb.ParameterSchema {
	return &toolspb.ParameterSchema{
		Type:        "string",
		Description: description,
		Format:      format,
	}
}

// IntParam creates an integer parameter schema.
func IntParam(description string) *toolspb.ParameterSchema {
	return &toolspb.ParameterSchema{
		Type:        "integer",
		Description: description,
	}
}

// IntParamWithDefault creates an integer parameter schema with a default value.
func IntParamWithDefault(description string, defaultVal int) *toolspb.ParameterSchema {
	return &toolspb.ParameterSchema{
		Type:        "integer",
		Description: description,
		Default:     IntValue(defaultVal),
	}
}

// BoolParam creates a boolean parameter schema.
func BoolParam(description string) *toolspb.ParameterSchema {
	return &toolspb.ParameterSchema{
		Type:        "boolean",
		Description: description,
	}
}

// BoolParamWithDefault creates a boolean parameter schema with a default value.
func BoolParamWithDefault(description string, defaultVal bool) *toolspb.ParameterSchema {
	return &toolspb.ParameterSchema{
		Type:        "boolean",
		Description: description,
		Default:     BoolValue(defaultVal),
	}
}

// ObjectParam creates an object parameter schema.
func ObjectParam(description string) *toolspb.ParameterSchema {
	return &toolspb.ParameterSchema{
		Type:        "object",
		Description: description,
	}
}

// ArrayParam creates an array parameter schema with string items.
func ArrayParam(description string, itemType string) *toolspb.ParameterSchema {
	return &toolspb.ParameterSchema{
		Type:        "array",
		Description: description,
		Items: &toolspb.ParameterSchema{
			Type: itemType,
		},
	}
}

// EnumParam creates a string parameter with enum values.
func EnumParam(description string, values []string) *toolspb.ParameterSchema {
	return &toolspb.ParameterSchema{
		Type:        "string",
		Description: description,
		Enum:        values,
	}
}

// BoolValue creates a JsonValue from a bool.
func BoolValue(v bool) *commonv1.JsonValue {
	return &commonv1.JsonValue{
		Kind: &commonv1.JsonValue_BoolValue{BoolValue: v},
	}
}

// IntValue creates a JsonValue from an int.
func IntValue(v int) *commonv1.JsonValue {
	return &commonv1.JsonValue{
		Kind: &commonv1.JsonValue_IntValue{IntValue: int64(v)},
	}
}

// StringValue creates a JsonValue from a string.
func StringValue(v string) *commonv1.JsonValue {
	return &commonv1.JsonValue{
		Kind: &commonv1.JsonValue_StringValue{StringValue: v},
	}
}

// NewToolParameters creates a new ToolParameters struct with the given properties.
func NewToolParameters(props map[string]*toolspb.ParameterSchema, required []string) *toolspb.ToolParameters {
	return &toolspb.ToolParameters{
		Type:       "object",
		Properties: props,
		Required:   required,
	}
}

// NewToolMetadata creates a new ToolMetadata with common defaults.
func NewToolMetadata(opts ToolMetadataOpts) *toolspb.ToolMetadata {
	meta := &toolspb.ToolMetadata{
		EnabledByDefault:   opts.EnabledByDefault,
		RequiresApproval:   opts.RequiresApproval,
		TimeoutSeconds:     opts.TimeoutSeconds,
		RateLimitPerMinute: opts.RateLimitPerMinute,
		CostEstimate:       opts.CostEstimate,
		LongRunning:        opts.LongRunning,
		Idempotent:         opts.Idempotent,
		ModifiesState:      opts.ModifiesState,
		SensitiveOutput:    opts.SensitiveOutput,
		Tags:               opts.Tags,
	}

	// Set defaults
	if meta.TimeoutSeconds == 0 {
		meta.TimeoutSeconds = 30
	}
	if meta.CostEstimate == "" {
		meta.CostEstimate = "low"
	}

	return meta
}

// ToolMetadataOpts are options for creating tool metadata.
type ToolMetadataOpts struct {
	EnabledByDefault   bool
	RequiresApproval   bool
	TimeoutSeconds     int32
	RateLimitPerMinute int32
	CostEstimate       string // "low", "medium", "high", "variable"
	LongRunning        bool
	Idempotent         bool
	ModifiesState      bool
	SensitiveOutput    bool
	Tags               []string
}

// NewAsyncBehavior creates a new AsyncBehavior for long-running tools.
func NewAsyncBehavior(opts AsyncBehaviorOpts) *toolspb.AsyncBehavior {
	behavior := &toolspb.AsyncBehavior{
		StatusPolling: &toolspb.StatusPolling{
			StatusTool:             opts.StatusTool,
			OperationIdField:       opts.OperationIdField,
			StatusToolIdParam:      opts.StatusToolIdParam,
			PollIntervalSeconds:    opts.PollIntervalSeconds,
			MaxPollDurationSeconds: opts.MaxPollDurationSeconds,
		},
		CompletionConditions: &toolspb.CompletionConditions{
			StatusField:   opts.StatusField,
			SuccessValues: opts.SuccessValues,
			FailureValues: opts.FailureValues,
			PendingValues: opts.PendingValues,
			ErrorField:    opts.ErrorField,
			ResultField:   opts.ResultField,
		},
	}

	// Set defaults
	if behavior.StatusPolling.PollIntervalSeconds == 0 {
		behavior.StatusPolling.PollIntervalSeconds = 5
	}
	if behavior.StatusPolling.MaxPollDurationSeconds == 0 {
		behavior.StatusPolling.MaxPollDurationSeconds = 3600 // 1 hour
	}

	// Add cancellation if specified
	if opts.CancelTool != "" {
		behavior.Cancellation = &toolspb.CancellationBehavior{
			CancelTool:        opts.CancelTool,
			CancelToolIdParam: opts.CancelToolIdParam,
			Graceful:          true,
		}
	}

	return behavior
}

// AsyncBehaviorOpts are options for creating async behavior.
type AsyncBehaviorOpts struct {
	// Status polling
	StatusTool             string
	OperationIdField       string
	StatusToolIdParam      string
	PollIntervalSeconds    int32
	MaxPollDurationSeconds int32

	// Completion conditions
	StatusField   string
	SuccessValues []string
	FailureValues []string
	PendingValues []string
	ErrorField    string
	ResultField   string

	// Cancellation (optional)
	CancelTool        string
	CancelToolIdParam string
}

// NewToolDefinition creates a new tool definition with the given parameters.
func NewToolDefinition(name, description, category string, params *toolspb.ToolParameters, metadata *toolspb.ToolMetadata) *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        name,
		Description: description,
		Category:    category,
		Parameters:  params,
		Metadata:    metadata,
	}
}

// NewCategory creates a new tool category.
func NewCategory(id, name, description, icon string, order int32) *toolspb.ToolCategory {
	return &toolspb.ToolCategory{
		Id:           id,
		Name:         name,
		Description:  description,
		Icon:         icon,
		DisplayOrder: order,
	}
}
