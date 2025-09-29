# Test Genie Initialization

This directory contains initialization scripts and configurations for the Test Genie scenario.

## Directory Structure

```
initialization/
├── README.md                 # This file
├── storage/                  # Database and storage setup
│   ├── postgres/            # PostgreSQL initialization
│   │   ├── schema.sql       # Database schema
│   │   └── seed.sql         # Initial data
│   └── redis/               # Redis cache setup (optional)
├── configuration/           # Configuration files
│   ├── defaults.json        # Default settings
│   └── test-patterns.yaml   # Test generation patterns
└── scripts/                 # Setup and maintenance scripts
    ├── setup.sh            # Main setup script
    └── health-check.sh     # Health check script
```

## Setup Process

1. **Database Initialization**: Creates required tables and indexes
2. **Configuration Loading**: Sets up default configurations
3. **Pattern Registration**: Loads test generation patterns
4. **Health Verification**: Verifies all components are working

## Dependencies

- PostgreSQL 12+ (required)
- App Issue Tracker (required for delegated test generation)
- Redis (optional, for caching)
- Node.js 16+ (for UI)

## Environment Variables

See the main README.md for complete environment variable documentation.
