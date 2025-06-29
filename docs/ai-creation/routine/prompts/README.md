# Routine Generation Prompts

This directory contains specialized prompts for generating different types of Vrooli routines. Each prompt is tailored to create high-quality, production-ready routines for specific use cases.

## Prompt Index

### Multi-Step Routines
- **[multi-step-prompt.md](./multi-step-prompt.md)** - Complex workflows with BPMN or Sequential execution
  - Use for: Business processes, orchestrated workflows, multi-agent coordination
  - Strategy: Conversational, Reasoning, or Deterministic
  - Structure: RoutineMultiStep with graph-based execution

### Single-Step Routines

#### AI-Powered Operations
- **[single-step-generate-prompt.md](./single-step-generate-prompt.md)** - LLM text generation and analysis
  - Use for: Content creation, analysis, reasoning, conversation
  - Strategy: Conversational, Reasoning, or Deterministic
  - Structure: RoutineGenerate with callDataGenerate

#### External Integrations
- **[single-step-api-prompt.md](./single-step-api-prompt.md)** - REST API calls to external services
  - Use for: Third-party integrations, data retrieval, webhooks
  - Strategy: Deterministic (recommended)
  - Structure: RoutineApi with callDataApi

- **[single-step-web-prompt.md](./single-step-web-prompt.md)** - Web search and information retrieval
  - Use for: Research, fact-checking, current information
  - Strategy: Deterministic or Reasoning
  - Structure: RoutineWeb with callDataWeb

#### Data Processing
- **[single-step-code-prompt.md](./single-step-code-prompt.md)** - Sandboxed code execution
  - Use for: Calculations, algorithms, data transformation
  - Strategy: Deterministic (recommended)
  - Structure: RoutineCode with callDataCode

- **[single-step-data-prompt.md](./single-step-data-prompt.md)** - Specialized data processing
  - Use for: ETL, validation, format conversion, aggregation
  - Strategy: Deterministic
  - Structure: RoutineData with callDataData

#### Platform Operations
- **[single-step-action-prompt.md](./single-step-action-prompt.md)** - MCP tool calls for platform actions
  - Use for: Resource management, notifications, system operations
  - Strategy: Deterministic (recommended)
  - Structure: RoutineInternalAction with callDataAction

## Usage Guidelines

### When to Use Each Prompt Type

#### Start with Multi-Step for:
- Complex business processes
- Workflows requiring multiple operations
- Orchestrated agent interactions
- Decision trees and branching logic

#### Use Single-Step for Subroutines:
- **Generate**: AI text processing, analysis, content creation
- **Api**: External service integrations, data fetching
- **Code**: Mathematical calculations, algorithmic operations
- **Web**: Research, information gathering, fact-checking
- **Data**: ETL processes, data validation, format conversion
- **Action**: Platform resource management, user operations

### Generation Workflow

1. **Identify Main Routine Type**: Start with the overall workflow structure
2. **Analyze Subroutine Needs**: Identify atomic operations required
3. **Classify Subroutines**: Determine the appropriate single-step type for each
4. **Generate Subroutines First**: Create atomic operations as separate routines
5. **Generate Main Routine**: Create the orchestrating workflow with proper references

### Quality Standards

All prompts enforce these quality standards:
- **Complete JSON Structure**: Valid database-compatible format
- **Proper ID Generation**: 19-digit snowflake IDs and 10-12 char publicIds
- **Template Variables**: Correct `{{variable}}` syntax
- **Form Configuration**: Complete input/output form definitions
- **Strategy Alignment**: Appropriate execution strategy selection
- **Error Handling**: Robust error handling and validation
- **Documentation**: Clear names, descriptions, and usage instructions

### Validation Requirements

Each generated routine must pass:
- **Structural Validation**: JSON schema compliance
- **ID Format Validation**: Correct snowflake and publicId formats
- **Template Validation**: Proper variable syntax and mapping
- **Form Validation**: Complete input/output form schemas
- **Strategy Validation**: Appropriate execution strategy selection
- **Content Validation**: Clear documentation and instructions

## Integration with Generation Scripts

These prompts are used by the enhanced generation system:

1. **Type Classification**: Scripts analyze subroutine needs and select appropriate prompts
2. **Multi-Pass Generation**: Generate subroutines first, then main routines
3. **Smart Resolution**: Reuse existing routines when possible
4. **Hierarchical Import**: Import atomic operations before complex workflows

## Customization Guidelines

When creating new prompts or modifying existing ones:

### Follow the Template Structure:
1. **Overview**: Purpose and use cases
2. **Execution Context**: Tier integration and workflow usage
3. **JSON Structure**: Complete routine format
4. **Configuration Details**: Specific config options
5. **Common Use Cases**: Templates and examples
6. **Form Configuration**: Input/output form patterns
7. **Validation Rules**: Quality requirements
8. **Generation Guidelines**: Best practices
9. **Quality Checklist**: Pre-generation verification

### Maintain Consistency:
- Use consistent terminology across prompts
- Follow the same validation patterns
- Include comprehensive examples
- Document all configuration options
- Provide clear generation guidelines

### Test Thoroughly:
- Generate sample routines with each prompt
- Validate against database schema
- Test import and execution
- Verify workflow integration
- Check error handling scenarios

## Contributing

When adding new routine types or improving existing prompts:

1. **Follow the established template structure**
2. **Include comprehensive examples and use cases**
3. **Test generated routines thoroughly**
4. **Update this README with new prompt information**
5. **Ensure compatibility with generation scripts**
6. **Document any new validation requirements**

## Related Documentation

- [../README.md](../README.md) - Main routine creation system overview
- [../validate-routine.sh](../validate-routine.sh) - Validation script
- [../../architecture/execution/tiers/tier2-process-intelligence/routine-types.md](../../architecture/execution/tiers/tier2-process-intelligence/routine-types.md) - Routine architecture overview