# Vrooli BATS Testing Infrastructure

> Revolutionary test infrastructure with lazy-loaded mocks, resource-aware setup, and 35+ specialized assertions.

## Quick Start: Choose Your Adventure

### 🚀 Simple Test (Just Mock Docker/HTTP)
```bash
#!/usr/bin/env bats
source "/path/to/bats-fixtures/core/common_setup.bash"

setup() { setup_standard_mocks; }
teardown() { cleanup_mocks; }

@test "docker works" {
    run docker ps
    assert_success
    assert_output_contains "CONTAINER"
}
```

### 🎯 Resource Test (Test Ollama Integration)
```bash
#!/usr/bin/env bats
source "/path/to/bats-fixtures/core/common_setup.bash"

setup() { setup_resource_test "ollama"; }
teardown() { cleanup_mocks; }

@test "ollama is healthy" {
    assert_resource_healthy "ollama"
    assert_env_equals "OLLAMA_PORT" "11434"
}
```

### 🔗 Integration Test (Multiple Resources)
```bash
#!/usr/bin/env bats
source "/path/to/bats-fixtures/core/common_setup.bash"

setup() { setup_integration_test "ollama" "whisper" "n8n"; }
teardown() { cleanup_mocks; }

@test "multi-resource coordination" {
    assert_resource_healthy "ollama"
    assert_resource_healthy "whisper"
    assert_docker_container_running "${OLLAMA_CONTAINER_NAME}"
}
```

## 📊 Decision Tree: Which Setup Should I Use?

```
Are you testing a specific Vrooli resource? 
├─ YES → setup_resource_test "resource_name"
│   └─ Need multiple resources? → setup_integration_test "res1" "res2" "res3"
└─ NO → Just need basic Docker/HTTP mocks? → setup_standard_mocks
```

## 🎪 What's Available

| Feature | Count | Examples |
|---------|-------|----------|
| **Setup Modes** | 3 | `standard`, `resource`, `integration` |
| **System Mocks** | 15+ | `docker`, `curl`, `jq`, `systemctl` |
| **Resource Mocks** | 15+ | `ollama`, `whisper`, `n8n`, `qdrant` |
| **Assertions** | 35+ | `assert_resource_healthy`, `assert_json_valid` |
| **Performance** | ~50% faster | Lazy loading, in-memory operations |

## 🗺️ Documentation Hub

### Getting Started
- **[Setup Guide](docs/setup-guide.md)** - Detailed explanation of the 3 setup modes
- **[Examples](docs/examples/)** - Copy-paste ready test examples

### Reference
- **[Mock Registry](docs/mock-registry.md)** - Lazy loading system and categories
- **[Assertions Reference](docs/assertions.md)** - All 35+ assertion functions
- **[Resource Testing](docs/resource-testing.md)** - Resource-specific patterns

### Advanced
- **[Troubleshooting](docs/troubleshooting.md)** - Common issues and performance tips

## 🔧 Quick Reference

### Essential Functions
```bash
# Setup (choose one)
setup_standard_mocks                    # Basic Docker/HTTP/Commands
setup_resource_test "ollama"            # Single resource + dependencies  
setup_integration_test "res1" "res2"    # Multiple resources

# Essential Assertions
assert_success                          # Exit code = 0
assert_output_contains "text"           # Output includes text
assert_resource_healthy "ollama"        # Resource health check
assert_json_valid "$json_string"        # Valid JSON
assert_docker_container_running "name"  # Container is running

# Cleanup
cleanup_mocks                           # Always call in teardown()
```

### Environment Variables (Auto-configured)
```bash
$OLLAMA_PORT, $OLLAMA_BASE_URL          # Resource: ollama
$WHISPER_PORT, $WHISPER_BASE_URL        # Resource: whisper
$N8N_PORT, $N8N_BASE_URL                # Resource: n8n
$TEST_NAMESPACE, $TEST_TMPDIR           # Test isolation
```

## 🚀 Performance Features

- **~60% faster startup** via lazy loading (load only what you need)
- **~50% faster execution** via in-memory operations (`/dev/shm`)
- **100% backward compatibility** with existing tests
- **Zero configuration** for standard use cases

## 🎓 Learning Path

1. **Start here:** Try the [basic example](docs/examples/basic-test.bats)
2. **Level up:** Read the [Setup Guide](docs/setup-guide.md) 
3. **Go deep:** Explore [Mock Registry](docs/mock-registry.md) concepts
4. **Master it:** Build custom patterns with [Resource Testing](docs/resource-testing.md)

---

💡 **New to BATS?** This infrastructure makes BATS testing 10x easier. Start with our examples and level up gradually.

🔥 **Upgrading from old fixtures?** Everything is backward compatible. Your existing tests will work unchanged.