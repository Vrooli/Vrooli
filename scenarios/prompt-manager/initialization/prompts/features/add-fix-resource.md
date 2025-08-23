# Resource Development and Enhancement

You are tasked with adding or fixing a resource in Vrooli's local resource ecosystem. Resources are the foundation of Vrooli's capabilities - each new resource exponentially increases what scenarios can accomplish.

## Understanding Resources

**What Resources Are**: Local services that extend Vrooli's capabilities through modular, orchestrated components. Resources include AI inference (Ollama), automation platforms (n8n), databases (PostgreSQL), and specialized services (Agent-S2 for desktop automation).

**Why This Matters**: Every resource added multiplies the capabilities of ALL scenarios. A new crypto wallet resource gives financial capabilities. A physics simulation resource enables research scenarios. Resources are Vrooli's superpower multiplier.

## Pre-Implementation Research

**Essential Reading**: Study interface standards at `scripts/resources/docs/interface-standards.md` and examine existing resources in `scripts/resources/`. Use `tree` commands to analyze structure patterns for similar resource categories.

## Required Resource Structure

Every resource must implement this exact structure:

```
scripts/resources/{category}/{resource-name}/
├── README.md                    # Complete resource documentation
├── manage.sh                    # Main interface (install, start, stop, status, logs)
├── cli.sh                       # Command-line interface
├── config/
│   ├── defaults.sh              # Default configuration
│   ├── messages.sh              # User-facing messages
│   └── capabilities.yaml        # Resource capabilities
├── lib/
│   ├── common.sh                # Shared utilities
│   ├── install.sh               # Installation logic
│   ├── status.sh                # Status checking
│   └── [specific functions]     # Resource-specific functionality
├── inject.sh                    # Scenario injection adapter
├── test/
│   └── integration-test.sh      # Integration tests
└── docs/                        # Additional documentation
```

## Implementation Phases

### Phase 1: Setup & Configuration
- Reserve port via `./scripts/resources/port_registry.sh --action list`
- Create directory structure: `mkdir -p scripts/resources/{category}/{resource-name}/{config,lib,test,docs}`
- Configure defaults, messages, and capabilities in `config/` folder

### Phase 2: Core Implementation
- Implement `manage.sh` with required actions: install, start, stop, status, logs
- Create utility functions in `lib/` (common.sh, install.sh, status.sh)
- Build resource-specific functionality

### Phase 3: Integration & Testing
- Create injection adapter (`inject.sh`) following existing patterns
- Implement integration tests covering installation, lifecycle, API, and error conditions
- Update resource index and port registry

### Phase 4: Validation & Documentation
- Run validation: `./scripts/resources/tools/validate-interfaces.sh --resource {name}`
- Complete README with API docs and integration examples
- Test end-to-end with actual scenarios

## Validation and Testing

### 1. Interface Compliance
```bash
# Validate interface standards
./scripts/resources/tools/validate-interfaces.sh --level quick --resource {resource-name}
./scripts/resources/tools/validate-interfaces.sh --level standard --resource {resource-name}
./scripts/resources/tools/validate-interfaces.sh --level full --resource {resource-name}
```

### 2. Three-Layer Testing
- **Layer 1** (< 1 second): Syntax validation
- **Layer 2** (< 30 seconds): Behavioral testing  
- **Layer 3** (< 5 minutes): Integration testing

### 3. Port Registry Validation
```bash
./scripts/resources/port_registry.sh --action validate
```

## Integration Testing

### 1. Resource Index Integration
Update `scripts/resources/index.sh` to include your resource.

### 2. Service Configuration
```json
{ "resources": { "your-resource": { "enabled": true } } }
```

### 3. End-to-End Testing
```bash
# Test complete lifecycle
./scripts/manage.sh setup --resources "your-resource"
./scripts/resources/index.sh --action status --resources "your-resource"
```

## Documentation Requirements

### 1. Resource README.md
Include:
- Clear purpose and capabilities
- Installation instructions
- API documentation
- Integration examples
- Troubleshooting guide

### 2. Update Global Documentation
- Add resource to `scripts/resources/README.md`
- Update capability matrix
- Add integration examples

## Success Criteria

### ✅ Resource Implementation
- [ ] Complete directory structure
- [ ] All required interface actions work
- [ ] Passes all three validation layers
- [ ] Port properly registered and functional
- [ ] Injection system working

### ✅ Integration
- [ ] Resource discoverable via index.sh
- [ ] Configurable via service.json
- [ ] Works with main lifecycle management
- [ ] Ready for scenario injection

### ✅ Testing
- [ ] Integration tests pass
- [ ] No port conflicts
- [ ] Clean installation/removal
- [ ] Error handling working

### ✅ Documentation
- [ ] Complete README with examples
- [ ] API documentation
- [ ] Integration patterns documented
- [ ] Added to global resource catalog

## Common Pitfalls to Avoid

1. **Port Conflicts**: Always check port registry first
2. **Interface Violations**: Follow interface standards exactly
3. **Missing Injection**: Scenarios won't work without proper injection adapter
4. **Inadequate Testing**: All three validation layers must pass
5. **Poor Documentation**: Resources are useless if others can't understand/use them

## File Locations Reference

- **Interface Standards**: `scripts/resources/docs/interface-standards.md`
- **Port Registry**: `scripts/resources/port_registry.sh`  
- **Resource Index**: `scripts/resources/index.sh`
- **Validation Tools**: `scripts/resources/tools/`
- **Example Resources**: `scripts/resources/{category}/`

## Next Steps After Implementation

1. **Create Sample Scenario**: Build a scenario that showcases your resource
2. **Integration Examples**: Document how other resources can work with yours
3. **Performance Testing**: Ensure resource scales appropriately
4. **Security Review**: Validate security implications

Remember: Each resource you add makes EVERY future scenario more powerful. Build with quality and foresight.