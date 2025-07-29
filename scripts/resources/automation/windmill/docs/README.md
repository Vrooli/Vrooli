# Windmill Documentation

Welcome to the Windmill documentation. This guide is organized into focused sections to help you find exactly what you need.

## 📚 Documentation Structure

### Getting Started
- [Installation Guide](INSTALLATION.md) - Setup and installation instructions
- [Configuration](CONFIGURATION.md) - Environment variables and settings
- [Quick Start Examples](../examples/README.md) - Jump right into using Windmill

### Core Concepts
- [Architecture Overview](ARCHITECTURE.md) - Services, storage, and system design
- [Scripts vs Flows vs Apps](CONCEPTS.md) - Understanding Windmill's building blocks
- [App Management](app-management.md) - Creating and managing Windmill apps

### Development
- [API Reference](API.md) - REST API endpoints and usage
- [Language Support](LANGUAGES.md) - TypeScript, Python, Go, and Bash guides
- [Workflows Guide](../examples/flows/README.md) - Building complex workflows

### Operations
- [Worker Management](OPERATIONS.md#worker-management) - Scaling and managing workers
- [Backup & Restore](OPERATIONS.md#backup-restore) - Data protection procedures
- [Monitoring](OPERATIONS.md#monitoring) - Health checks and performance tracking

### Troubleshooting
- [Common Issues](TROUBLESHOOTING.md) - Solutions to frequent problems
- [Performance Tuning](PERFORMANCE.md) - Optimization tips
- [FAQ](FAQ.md) - Frequently asked questions

## 🚀 Quick Links

- **Web Interface**: http://localhost:5681
- **API Documentation**: http://localhost:5681/openapi.html
- **Default Credentials**: admin@windmill.dev / changeme

## 📋 Quick Reference

```bash
# Install Windmill
./manage.sh --action install

# Check status
./manage.sh --action status

# View logs
./manage.sh --action logs

# Scale workers
./manage.sh --action scale-workers --workers 5
```

## 🔗 External Resources

- [Official Windmill Documentation](https://www.windmill.dev/docs)
- [Windmill GitHub Repository](https://github.com/windmill-labs/windmill)
- [Community Discord](https://discord.com/invite/V7PM2YHsPB)