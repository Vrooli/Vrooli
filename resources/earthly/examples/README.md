# Earthly Examples

This directory contains example Earthfiles demonstrating various Earthly features.

## Simple Example

The `simple/Earthfile` demonstrates:
- Basic build targets
- Test execution
- Artifact saving
- Parallel execution
- Multi-platform builds
- Layer caching

### Running the Examples

```bash
# Execute test target
vrooli resource earthly content execute --file=examples/simple/Earthfile --target=+test

# Execute build with caching
vrooli resource earthly content execute --file=examples/simple/Earthfile --target=+build --cache

# Execute parallel demo
vrooli resource earthly content execute --file=examples/simple/Earthfile --target=+parallel-demo

# Multi-platform build
vrooli resource earthly content execute --file=examples/simple/Earthfile --target=+multi-platform --platform=linux/amd64,linux/arm64
```

## Advanced Examples

More complex examples can be added here demonstrating:
- CI/CD integration
- Monorepo builds
- Docker image creation
- Cross-repository dependencies
- Secret management
- Remote caching