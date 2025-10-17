# OpenMRS Clinical Records Platform

OpenMRS is an open-source electronic medical record (EMR) platform designed to support healthcare delivery in resource-constrained environments. This resource integrates OpenMRS into the Vrooli ecosystem, providing REST and FHIR APIs for healthcare scenarios.

## Features

- **Complete EMR System**: Patient management, encounters, clinical concepts, and provider tracking
- **REST API**: Comprehensive RESTful API for all clinical operations
- **FHIR Support**: HL7 FHIR R4 API for healthcare interoperability
- **MySQL Backend**: Robust MySQL database for clinical data storage
- **Docker Deployment**: Containerized for easy deployment and scaling
- **Demo Data**: CLI command to seed sample patients and clinical data

## Important Note

OpenMRS requires initial configuration on first run. After starting the service, navigate to http://localhost:8005/openmrs to complete the setup wizard. The REST API will be available after initial configuration is complete.

## Quick Start

```bash
# Install OpenMRS
vrooli resource openmrs manage install

# Start services
vrooli resource openmrs manage start --wait

# Check status
vrooli resource openmrs status

# Run health checks
vrooli resource openmrs test smoke
```

## Access Points

- **Web Interface**: http://localhost:8005/openmrs
- **REST API**: http://localhost:8005/openmrs/ws/rest/v1
- **FHIR API**: http://localhost:8005/openmrs/ws/fhir2/R4
- **Default Login**: admin / Admin123
- **Database**: MySQL on port 3316

## CLI Commands

### Lifecycle Management
```bash
# Install and configure
vrooli resource openmrs manage install

# Start services
vrooli resource openmrs manage start [--wait]

# Stop services
vrooli resource openmrs manage stop

# Restart services
vrooli resource openmrs manage restart

# Uninstall (optionally keep data)
vrooli resource openmrs manage uninstall [--keep-data]
```

### Demo Data
```bash
# Import demo patients and data
vrooli resource openmrs content execute import-demo-data
```

### Patient Management
```bash
# List all patients
vrooli resource openmrs content add patient "John" "Smith" "M" "1985-03-15"

# Get specific patient
vrooli resource openmrs content get patient <uuid>

# List all patients (via content system)
vrooli resource openmrs content execute patient list
```

### Encounter Management
```bash
# List encounters
vrooli resource openmrs api encounter list

# Create encounter (coming soon)
vrooli resource openmrs api encounter create --patient-id <uuid> --type "consultation"
```

### FHIR Operations
```bash
# Export FHIR data (coming soon)
vrooli resource openmrs fhir export --resource-type "Patient"

# Import FHIR bundle (coming soon)
vrooli resource openmrs fhir import --file bundle.json
```

### Testing
```bash
# Run all tests
vrooli resource openmrs test all

# Quick health check
vrooli resource openmrs test smoke

# API integration tests
vrooli resource openmrs test integration

# Unit tests
vrooli resource openmrs test unit
```

## API Examples

### REST API Authentication
```bash
# Get session
curl -u admin:Admin123 \
  http://localhost:8006/openmrs/ws/rest/v1/session
```

### List Patients
```bash
# Get all patients
curl -u admin:Admin123 \
  http://localhost:8006/openmrs/ws/rest/v1/patient
```

### Get Patient by UUID
```bash
# Get specific patient
curl -u admin:Admin123 \
  http://localhost:8006/openmrs/ws/rest/v1/patient/{uuid}
```

### List Encounters
```bash
# Get all encounters
curl -u admin:Admin123 \
  http://localhost:8006/openmrs/ws/rest/v1/encounter
```

## Configuration

Configuration is managed through environment variables:

```bash
# Ports
OPENMRS_PORT=8005           # Web interface
OPENMRS_API_PORT=8006       # REST API
OPENMRS_FHIR_PORT=8007      # FHIR API
OPENMRS_DB_PORT=5444        # PostgreSQL

# Credentials
OPENMRS_ADMIN_USER=admin
OPENMRS_ADMIN_PASS=Admin123

# Features
OPENMRS_ENABLE_FHIR=true
OPENMRS_ENABLE_DEMO_DATA=true
```

## Integration with Vrooli

OpenMRS can be integrated with other Vrooli resources:

- **Ollama/Haystack**: Clinical decision support and documentation assistance
- **n8n**: Automated workflows for referrals and notifications
- **Apache Superset**: Population health dashboards
- **Qdrant**: Semantic search over clinical knowledge
- **PostgreSQL**: Shared database infrastructure
- **Keycloak**: Enterprise authentication (coming soon)

## Architecture

```
┌─────────────────────────┐
│   OpenMRS Web UI        │
│   (Port 8005)           │
└─────────┬───────────────┘
          │
┌─────────▼───────────────┐
│   OpenMRS Backend       │
├─────────────────────────┤
│ • REST API (8006)       │
│ • FHIR API (8007)       │
│ • Business Logic        │
└─────────┬───────────────┘
          │
┌─────────▼───────────────┐
│   PostgreSQL Database   │
│   (Port 5444)           │
│ • Patient Records       │
│ • Clinical Data         │
│ • Audit Logs            │
└─────────────────────────┘
```

## Data Model

### Core Entities
- **Patient**: Demographics, identifiers, contact information
- **Encounter**: Clinical visits and interactions
- **Observation**: Vital signs, lab results, clinical findings
- **Concept**: Clinical terminology and coding systems
- **Provider**: Healthcare professionals
- **Location**: Facilities and departments
- **Order**: Prescriptions, lab orders, procedures

## Security Considerations

- **Authentication**: Basic auth for API, session-based for UI
- **Authorization**: Role-based access control (RBAC)
- **Audit Logging**: All data access is logged
- **Data Encryption**: TLS for API communication (configure reverse proxy)
- **HIPAA Compliance**: Configurable for regulatory requirements

## Troubleshooting

### Service Won't Start
```bash
# Check Docker status
docker ps -a | grep openmrs

# View logs
vrooli resource openmrs logs app
vrooli resource openmrs logs db

# Check port availability
lsof -i :8005
```

### API Not Responding
```bash
# Check container health
docker inspect openmrs-app | jq '.[0].State.Health'

# Test connectivity
curl -I http://localhost:8005/openmrs
```

### Database Issues
```bash
# Check PostgreSQL status
docker exec openmrs-postgres pg_isready

# View database logs
docker logs openmrs-postgres
```

## Performance Optimization

- **Initial Startup**: First start takes 2-3 minutes to initialize database
- **Memory**: Allocate at least 4GB RAM for optimal performance
- **Disk**: Reserve 10GB for data growth
- **Caching**: Enable application-level caching for better response times

## Limitations

Current implementation is scaffolding with basic functionality:
- Demo data seeding not yet implemented
- Full CRUD operations for all entities in progress
- FHIR module configuration pending
- Keycloak integration planned
- Backup/restore utilities coming soon

## Contributing

This resource follows the Vrooli v2.0 contract. Key files:
- `cli.sh`: Main CLI interface
- `lib/core.sh`: Core functionality
- `lib/test.sh`: Test implementations
- `config/`: Configuration files
- `test/phases/`: Test scripts

## License

OpenMRS is licensed under the Mozilla Public License 2.0. This Vrooli integration maintains the same license for OpenMRS components while the integration code follows Vrooli's licensing.

## Support

- **OpenMRS Documentation**: https://wiki.openmrs.org
- **FHIR Resources**: https://www.hl7.org/fhir/
- **Vrooli Integration**: See Vrooli documentation

## Next Steps

1. Complete demo data seeding
2. Implement full CRUD operations for all entities
3. Configure FHIR module
4. Add Keycloak authentication
5. Create n8n workflow templates
6. Build Superset dashboards
7. Implement backup/restore utilities