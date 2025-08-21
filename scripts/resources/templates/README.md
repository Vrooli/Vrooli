# Resource Templates

This directory contains templates for creating consistent and comprehensive resources in the Vrooli platform.

## Available Templates

### PRD Template (`PRD.md`)
Complete Product Requirements Document template for resources. This template ensures consistency across all Vrooli resources and prevents capability drift over time.

**Use this template when:**
- Creating a new resource
- Documenting an existing resource
- Ensuring consistent resource architecture
- Planning resource integration patterns

## Template Usage

```bash
# Copy template to your resource
cp scripts/resources/templates/PRD.md scripts/resources/storage/my-resource/PRD.md

# Edit the template to match your resource
# Fill in all sections completely
# Ensure integration patterns align with Vrooli architecture
```

## Template Philosophy

Resource PRDs differ from Scenario PRDs in important ways:

- **Resources** = Infrastructure components that enable scenarios
- **Scenarios** = Complete applications with business value

Resource PRDs focus on:
- Infrastructure capabilities and reliability
- Integration patterns with other resources  
- Operational concerns (deployment, monitoring, scaling)
- Resource-specific management interfaces
- System-level configuration and optimization

## Quality Standards

All resource PRDs should:
- ✅ Define clear integration patterns
- ✅ Specify operational requirements
- ✅ Document resource management interfaces
- ✅ Include comprehensive testing strategies
- ✅ Address security and compliance concerns
- ✅ Define resource lifecycle management