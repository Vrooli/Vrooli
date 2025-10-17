# Terraform Resource

HashiCorp Terraform infrastructure as code tool for declarative cloud resource provisioning.

## Quick Start

```bash
# Install and start Terraform
vrooli resource terraform manage install
vrooli resource terraform manage start

# Add configuration
vrooli resource terraform content add --file main.tf

# Run Terraform workflow
vrooli resource terraform content execute init
vrooli resource terraform content execute plan
vrooli resource terraform content execute apply

# Check status
vrooli resource terraform status
```

## Features

- **Multi-Cloud Support**: AWS, Azure, GCP, Kubernetes, 3000+ providers
- **State Management**: Secure state storage with locking and encryption
- **Plan/Apply Workflow**: Review changes before applying
- **Workspace Isolation**: Multiple environments from same configuration
- **HCL Language**: Human-readable infrastructure definitions

## Configuration

Default configuration in `config/defaults.sh`:
- Terraform version: 1.6.6
- State backend: local
- Auto-approve: disabled
- Parallelism: 10 operations

## Testing

```bash
# Quick health check
vrooli resource terraform test smoke

# Full test suite
vrooli resource terraform test all
```

## Integration

Works with:
- **Vault**: Secure credential storage
- **Git**: Version control for configurations
- **CI/CD**: Automated deployments
- **Monitoring**: Infrastructure change tracking

## Troubleshooting

### Common Issues

1. **Provider Plugin Download Fails**
   - Check network connectivity
   - Verify Docker has internet access
   - Try manual provider installation

2. **State Lock Timeout**
   - Another operation may be running
   - Check for stale locks
   - Force unlock if necessary

3. **Plan Takes Too Long**
   - Large infrastructure takes time
   - Increase timeout values
   - Use targeted plans

## Support

For issues or questions, check the PRD.md for detailed requirements and specifications.