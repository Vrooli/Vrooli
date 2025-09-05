# Test Data Generator

## Purpose
Standalone Node.js API service for generating realistic mock test data for development and testing purposes. Provides various data types, formats, and customizable schemas to accelerate development workflows across different projects.

## Usefulness to Vrooli
- **Testing Support**: Generates consistent, realistic test data for scenario validation
- **Development Acceleration**: Provides mock data for rapid prototyping and development
- **Data Variety**: Supports multiple data formats (JSON, CSV, SQL) and schemas
- **Cross-Scenario Utility**: Other scenarios can leverage this service for populating test databases

## Dependencies
### Required Resources
- **Node.js Runtime**: Primary execution environment
- **PostgreSQL**: Optional - for storing custom schemas and generated datasets
- **Redis**: Optional - for caching frequently requested data patterns

### Shared Workflows Used
- None initially - designed as a standalone service
- Future integration with n8n workflows for complex data generation patterns

## Architecture
### API (Node.js)
- RESTful endpoints for data generation
- Support for various data types (personal, business, financial, etc.)
- Schema-based generation with customizable parameters
- Bulk data generation capabilities

### CLI
- Command-line interface for local data generation
- Commands: `generate`, `schema`, `export`, `validate`
- Configuration file support

### UI
- Simple web interface for interactive data generation
- Schema builder with visual configuration
- Download capabilities for generated datasets
- Real-time preview of generated data

## UX Style
**Clean, functional interface** optimized for developer productivity with clear parameter controls, instant preview, and multiple export options.

## Key Features
1. **Multiple Data Types**: Personal info, addresses, companies, financial data, etc.
2. **Format Support**: JSON, CSV, XML, SQL inserts
3. **Schema Definition**: Custom data structures and relationships
4. **Bulk Generation**: Efficiently create large datasets
5. **Realistic Data**: Uses proper data patterns and formats
6. **Export Options**: Multiple download formats and direct API access
7. **Seeded Generation**: Consistent datasets for testing scenarios

## Usage by Other Scenarios
Other scenarios can interact with the Test Data Generator through:
- **API**: `POST /api/generate` with schema parameters
- **CLI**: `test-data-generator generate --type users --count 100`
- **Direct Integration**: Import as Node.js module for programmatic usage