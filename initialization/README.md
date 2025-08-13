# Shared Initialization Resources

This directory contains initialization data that can be shared across multiple scenarios, enabling reuse of common workflows, schemas, and configurations.

## Structure

```
initialization/
├── automation/
│   ├── n8n/                # Shared n8n workflows
│   │   └── embedding-generator.json
│   └── windmill/           # Shared Windmill apps (future)
├── storage/
│   ├── postgres/           # Shared database schemas
│   └── qdrant/            # Shared vector collections
└── configuration/         # Shared configs and templates
```

## Usage in Scenarios

To use shared resources in a scenario, reference them in your `.vrooli/service.json`:

```json
{
  "resources": {
    "automation": {
      "n8n": {
        "type": "n8n",
        "enabled": true,
        "initialization": [
          {
            "file": "shared:initialization/automation/n8n/embedding-generator.json",
            "type": "workflow"
          },
          {
            "file": "initialization/automation/n8n/scenario-specific-workflow.json", 
            "type": "workflow"
          }
        ]
      }
    }
  }
}
```

## Shared Resource Conventions

### File Naming
- Prefix shared resources with `shared-` 
- Use descriptive names: `embedding-generator.json`
- Version if needed: `embedding-generator-v2.json`

### Resource IDs
- Use globally unique IDs to prevent conflicts
- Include "shared" in the ID: `embedding-generator`

### Documentation
- Each shared resource should include inline documentation
- Add usage examples and expected input/output formats

## Benefits

1. **Reduced Duplication**: Common patterns implemented once
2. **Consistency**: Standardized implementations across scenarios
3. **Maintainability**: Update shared resources in one place
4. **Compound Intelligence**: Solutions become permanent capabilities

## Merge Behavior

When a scenario is converted to an app:
1. Shared resources are copied only if referenced in service.json
2. Scenario-specific resources take precedence over shared ones
3. Files are merged, not replaced - enabling customization
4. Both `shared:` prefix and direct path references are supported

## Contributing Shared Resources

When creating a shared resource:
1. Ensure it's truly reusable across multiple scenarios
2. Use generic, configurable implementations
3. Add comprehensive documentation
4. Test with multiple scenarios
5. Consider backward compatibility