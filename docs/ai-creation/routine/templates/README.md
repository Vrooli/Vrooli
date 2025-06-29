# Routine Templates Library

This directory contains reusable templates for common routine patterns. Templates provide starting points for routine generation and demonstrate best practices for each routine type.

## Template Organization

### By Routine Type
- **[api/](./api/)** - REST API integration templates
- **[code/](./code/)** - Code execution and data processing templates  
- **[generate/](./generate/)** - LLM text generation templates
- **[web/](./web/)** - Web search and research templates
- **[action/](./action/)** - Platform action and MCP tool templates
- **[data/](./data/)** - Data transformation and ETL templates

### Template Categories

#### **Basic Operations** - Simple, single-purpose routines
- CRUD operations for each routine type
- Standard data transformations
- Common API patterns

#### **Advanced Patterns** - Complex, feature-rich routines
- Multi-parameter operations
- Error handling and retry logic
- Performance-optimized implementations

#### **Integration Templates** - Workflow composition patterns
- Common subroutine combinations
- Data flow patterns between routine types
- Standard input/output mappings

## Using Templates

### As Generation Starting Points
Templates can be used to bootstrap routine generation:
1. Copy appropriate template JSON
2. Modify fields for specific use case
3. Update prompts and configurations
4. Validate and import

### As Reference Examples
Templates demonstrate:
- Proper JSON structure for each routine type
- Best practices for configuration
- Common patterns and approaches
- Error handling strategies

### In Generation Scripts
The enhanced generation script can reference templates:
- Template matching based on capability descriptions
- Automatic template selection for common patterns
- Template-based validation of generated routines

## Template Structure

Each template follows this structure:
```json
{
  "id": "template-id-placeholder",
  "publicId": "template-name", 
  "resourceType": "Routine",
  "isPrivate": false,
  "permissions": "{}",
  "isInternal": false,
  "tags": [],
  "versions": [
    {
      "id": "template-version-id",
      "publicId": "template-name-v1",
      "versionLabel": "1.0.0",
      "versionNotes": "Template description",
      "isComplete": true,
      "isPrivate": false,
      "versionIndex": 0,
      "isAutomatable": true,
      "resourceSubType": "[RoutineType]",
      "config": {
        "__version": "1.0",
        // Type-specific configuration
      },
      "translations": [
        {
          "id": "template-translation-id",
          "language": "en",
          "name": "Template Name",
          "description": "Template description and use case",
          "instructions": "How to use and customize this template"
        }
      ]
    }
  ]
}
```

## Template Naming Convention

Templates follow a consistent naming pattern:
- **File name**: `{category}-{function}-{variant}.json`
- **Public ID**: `{type}-{function}-{variant}-template`
- **Display name**: "{Category} {Function} {Variant}"

Examples:
- `api-rest-get-basic.json` → `api-get-basic-template` → "API Get Basic"
- `generate-analysis-sentiment.json` → `generate-sentiment-template` → "Generate Sentiment Analysis" 
- `code-transform-csv.json` → `code-csv-transform-template` → "Code CSV Transform"

## Quality Standards

All templates must meet these criteria:
- **Complete Structure**: Valid, importable JSON
- **Clear Documentation**: Comprehensive descriptions and instructions
- **Parameterized**: Use template variables where appropriate
- **Validated**: Pass structural and semantic validation
- **Tested**: Verified to work in development environment
- **Documented**: Include usage examples and customization notes

## Contributing Templates

When adding new templates:

### 1. Identify Common Patterns
- Review existing routines for repeated patterns
- Analyze backlog items for frequently requested capabilities
- Consider integration scenarios and workflow needs

### 2. Create Complete Templates
- Start with a working routine JSON
- Generalize specific values to template variables
- Add comprehensive documentation
- Include customization guidelines

### 3. Test Thoroughly
- Validate JSON structure
- Test import process
- Verify execution in development environment
- Check integration with other routine types

### 4. Document Usage
- Update this README with new template information
- Include clear usage instructions
- Provide customization examples
- Document any special requirements

## Template Usage in Generation

The enhanced generation system uses templates in several ways:

### **Pattern Matching**
- Compare capability descriptions to template patterns
- Suggest templates for common use cases
- Accelerate generation for standard operations

### **Quality Assurance**
- Validate generated routines against template patterns
- Ensure consistency across similar routine types
- Catch common configuration errors

### **Reusability**
- Promote consistent patterns across the platform
- Reduce generation time for common operations
- Improve reliability through tested patterns

## Related Documentation

- [../prompts/README.md](../prompts/README.md) - Generation prompt documentation
- [../README.md](../README.md) - Main routine creation system
- [../../architecture/execution/tiers/tier2-process-intelligence/routine-types.md](../../architecture/execution/tiers/tier2-process-intelligence/routine-types.md) - Routine architecture