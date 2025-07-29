# Windmill Workflow Automation Platform

Windmill is a developer-centric workflow automation platform that allows you to build workflows and UIs using code (TypeScript, Python, Go, Bash) instead of drag-and-drop interfaces.

## 🚀 Quick Start

```bash
# Install with default settings
./manage.sh --action install

# Check status
./manage.sh --action status

# Access web interface
open http://localhost:5681
```

**Default credentials**: admin@windmill.dev / changeme

## 📚 Documentation

For detailed information, see our comprehensive documentation:

- **[📖 Full Documentation](docs/README.md)** - Complete guide with all topics
- **[⚡ Installation Guide](docs/INSTALLATION.md)** - Setup and installation
- **[⚙️ Configuration](docs/CONFIGURATION.md)** - Environment variables and settings
- **[🏗️ Architecture](docs/ARCHITECTURE.md)** - System design and components
- **[💻 API Reference](docs/API.md)** - REST endpoints and usage

## 🎯 Core Features

- **Code-first workflows**: Write in TypeScript, Python, Go, or Bash
- **Built-in IDE**: Web-based editor with autocomplete
- **Scalable execution**: Multi-worker architecture
- **Real-time collaboration**: Multi-user with version control
- **Rich integrations**: 300+ pre-built connectors

## 🔧 Common Commands

```bash
# Service management
./manage.sh --action start|stop|restart|status

# Worker scaling
./manage.sh --action scale-workers --workers 5

# Monitoring
./manage.sh --action logs
./manage.sh --action backup

# Development
./manage.sh --action list-apps
./manage.sh --action import-examples
```

## 📋 System Requirements

- **Minimum**: 4GB RAM, 10GB disk space
- **Recommended**: 8GB RAM, 50GB disk space
- **Dependencies**: Docker, Docker Compose

## 🔗 Links

- **Web Interface**: http://localhost:5681
- **API Docs**: http://localhost:5681/openapi.html
- **Official Docs**: https://www.windmill.dev/docs
- **GitHub**: https://github.com/windmill-labs/windmill

---

**Need help?** Check the [troubleshooting guide](docs/TROUBLESHOOTING.md) or explore our [examples](examples/README.md).