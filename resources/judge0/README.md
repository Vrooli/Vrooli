# Judge0 Code Execution Resource

Secure sandboxed code execution supporting 60+ programming languages, enabling AI agents to validate generated code, run educational examples, and execute multi-language workflows.

## 🎯 Overview

Judge0 is the most advanced open-source code execution system, providing:
- **60+ Programming Languages**: JavaScript, Python, Go, Rust, Java, C++, and many more
- **Secure Sandboxing**: Multiple isolation layers with resource limits
- **Production Ready**: Powers competitive programming platforms worldwide
- **API-First Design**: RESTful API for seamless integration

## 🚀 Quick Start

### System Requirements

**Ubuntu 24.04 / Modern Linux**: Judge0's isolate sandbox requires unprivileged user namespaces, which are restricted by default on Ubuntu 24.04. During Vrooli setup with sudo, the following kernel parameter is automatically configured:

```bash
kernel.apparmor_restrict_unprivileged_userns = 0
```

This allows the isolate sandbox to create secure isolated environments for code execution. The setting is made persistent in `/etc/sysctl.d/99-vrooli.conf`.

### Installation
```bash
# Install Judge0 (auto-detects system and configures)
./resources/judge0/manage.sh --action install

# Install with custom settings
./resources/judge0/manage.sh --action install \
  --workers 4 \
  --cpu-limit 10 \
  --memory-limit 512
```

### Basic Usage
```bash
# Check status
./resources/judge0/manage.sh --action status

# Submit code
./resources/judge0/manage.sh --action submit \
  --code 'print("Hello, Judge0!")' \
  --language python

# List languages
./resources/judge0/manage.sh --action languages
```

## 🔧 Configuration

### Resource Limits (per submission)
```bash
CPU_TIME_LIMIT=5         # seconds
WALL_TIME_LIMIT=10       # seconds  
MEMORY_LIMIT=256         # MB
MAX_PROCESSES=30
MAX_FILE_SIZE=5          # MB
```

### Security Features
- **Network Isolation**: Disabled by default in execution containers
- **API Authentication**: Auto-generated secure token
- **Resource Limits**: Prevents infinite loops and memory bombs
- **Container Isolation**: Each submission runs in isolated environment

## 📚 Supported Languages

Judge0 supports 60+ languages. Here are some popular ones:

| Language | ID | Version | Example |
|----------|-----|---------|---------|
| JavaScript | 93 | Node.js 18.15.0 | `console.log("Hello");` |
| Python | 92 | 3.11.2 | `print("Hello")` |
| Go | 95 | 1.20.2 | `fmt.Println("Hello")` |
| Rust | 73 | 1.68.2 | `println!("Hello");` |
| Java | 91 | JDK 19.0.2 | `System.out.println("Hello");` |
| C++ | 105 | GCC 12.2.0 | `cout << "Hello";` |
| TypeScript | 94 | 5.0.3 | `console.log("Hello");` |

Run `./manage.sh --action languages` for the complete list.

## 🔗 Integration Examples

### Vrooli AI Tier Integration
```typescript
// Validate AI-generated code
const result = await judge0.executeCode({
  source_code: aiGeneratedCode,
  language_id: 93,  // JavaScript
  stdin: testInput,
  expected_output: expectedOutput
});

if (result.status.id === 3) {  // Accepted
  console.log("Code validated successfully!");
}
```

### Multi-Language Testing
```bash
# Test the same algorithm in multiple languages
for lang in javascript python go rust; do
  ./manage.sh --action submit \
    --code "@algorithm.$lang" \
    --language "$lang" \
    --stdin "test input"
done
```

### Batch Submissions
```javascript
// Submit multiple test cases
const submissions = testCases.map(test => ({
  source_code: code,
  language_id: 92,
  stdin: test.input,
  expected_output: test.output
}));

const results = await judge0.submitBatch(submissions);
```

## 🛡️ Security Best Practices

### 1. API Authentication
```bash
# API key is auto-generated during installation
# Located at: ~/.vrooli/resources/judge0/config/api_key

# Use in API calls:
curl -H "X-Auth-Token: YOUR_API_KEY" http://localhost:2358/submissions
```

### 2. Resource Limits
```bash
# Apply conservative limits for untrusted code
./manage.sh --action install \
  --cpu-limit 2 \
  --memory-limit 128 \
  --max-file-size 1
```

### 3. Network Isolation
Network access is disabled by default. Never enable for untrusted code:
```bash
ENABLE_NETWORK=false  # Keep this disabled
```

### 4. Security Validation
```bash
# Run security checks
./manage.sh --action security-validate

# Apply hardening
./manage.sh --action security-harden
```

## 📊 Monitoring & Management

### Status Information
```bash
# Comprehensive status
./manage.sh --action status

# Container statistics  
./manage.sh --action stats

# Resource usage
./manage.sh --action usage
```

### Logs
```bash
# View all logs
./manage.sh --action logs

# Server logs only
docker logs vrooli-judge0-server

# Worker logs
docker logs vrooli-judge0-workers-1
```

### Scaling Workers
```bash
# Scale to handle more concurrent submissions
./manage.sh --action scale-workers --count 8
```

## 🧪 Testing

### API Health Check
```bash
curl http://localhost:2358/system_info
```

### Simple Submission
```bash
curl -X POST http://localhost:2358/submissions?wait=true \
  -H "Content-Type: application/json" \
  -H "X-Auth-Token: YOUR_API_KEY" \
  -d '{
    "source_code": "console.log(\"Hello, World!\");",
    "language_id": 93
  }'
```

### Test All Languages
```bash
# Run comprehensive language test
./manage.sh --action test-languages
```

## 🚨 Troubleshooting

### Common Issues

**Judge0 not starting**
```bash
# Check port availability
lsof -i :2358

# Check Docker
docker ps -a | grep judge0

# View detailed logs
./manage.sh --action logs
```

**Submissions failing**
```bash
# Check worker status
docker ps | grep judge0-workers

# Verify API key
cat ~/.vrooli/resources/judge0/config/api_key

# Test with simple code
./manage.sh --action test
```

**Performance issues**
```bash
# Scale workers
./manage.sh --action scale-workers --count 4

# Check resource usage
./manage.sh --action stats

# Increase limits if needed
docker update --memory 4G vrooli-judge0-server
```

### Error Codes

| Status ID | Description | Solution |
|-----------|-------------|----------|
| 3 | Accepted | Success! |
| 4 | Wrong Answer | Check logic |
| 5 | Time Limit Exceeded | Optimize code or increase limit |
| 6 | Compilation Error | Fix syntax errors |
| 7-11 | Runtime Error | Check error output |
| 12 | Memory Limit Exceeded | Reduce memory usage |

## 🔄 Updates

```bash
# Check for updates
./manage.sh --action check-updates

# Update to latest version
./manage.sh --action update
```

## 📁 File Structure

```
~/.vrooli/resources/judge0/
├── config/
│   ├── api_key           # API authentication key
│   └── docker-compose.yml # Service configuration
├── logs/                 # Service logs
├── submissions/          # Submission storage
└── security-reports/     # Security audit logs
```

## 🔗 Advanced Usage

### Custom Language Configuration
```javascript
// Add custom compilation flags
{
  "language_id": 105,  // C++
  "source_code": code,
  "compiler_options": "-O2 -std=c++20",
  "command_line_arguments": "arg1 arg2"
}
```

### Callbacks (Advanced)
```javascript
// Get notified when execution completes
{
  "source_code": code,
  "language_id": 92,
  "callback_url": "https://your-api.com/webhook",
  "wait": false
}
```

### Performance Tuning
```bash
# For high-volume usage
export JUDGE0_WORKERS_COUNT=10
export JUDGE0_MAX_QUEUE_SIZE=1000
export JUDGE0_CPU_LIMIT=4
export JUDGE0_MEMORY_LIMIT_DOCKER=4G
```

## 📚 Resources

- **Official Docs**: https://judge0.com/docs
- **API Reference**: http://localhost:2358/docs
- **Examples**: `resources/judge0/examples/`
- **Security Guide**: `resources/judge0/docs/security.md`

## 🤝 Integration with Vrooli

Judge0 integrates seamlessly with Vrooli's three-tier AI system:

1. **Tier 1 (Coordination)**: Orchestrates code validation workflows
2. **Tier 2 (Process)**: Routes code execution tasks to Judge0
3. **Tier 3 (Execution)**: Direct API calls for immediate validation

Example workflow:
```
User Request → AI generates code → Judge0 validates → Results returned
```

This enables powerful capabilities like:
- AI code generation with automatic validation
- Educational coding tutorials with live execution
- Multi-language algorithm comparison
- Automated code testing and benchmarking

---

**Need help?** Run `./manage.sh --help` or check the [troubleshooting guide](#-troubleshooting).