# OpenEMR Resource

OpenEMR is a comprehensive open-source electronic medical records (EMR) and practice management system that provides healthcare organizations with clinical, administrative, and billing capabilities through modern REST and FHIR APIs.

## Quick Start

```bash
# Install and start OpenEMR
vrooli resource openemr manage install
vrooli resource openemr manage start --wait

# Verify health
vrooli resource openemr status

# Create a patient
vrooli resource openemr content add patient --name "John Doe" --dob "1980-01-01"

# Schedule appointment  
vrooli resource openemr content add appointment --patient 1 --provider 1 --date "2025-01-20" --time "10:00"

# Export data via FHIR
vrooli resource openemr content export fhir
```

## Features

- **Electronic Medical Records**: Complete patient records with medical history
- **Practice Management**: Scheduling, billing, and administrative tools
- **FHIR R4 Support**: Standards-based healthcare data exchange
- **REST API**: Modern API for custom integrations
- **Multi-Language**: Supports 30+ languages
- **Billing Integration**: Insurance claims and payment processing
- **Clinical Decision Support**: Alerts and reminders for providers
- **Patient Portal**: Secure access for patients to view records

## Architecture

OpenEMR runs as a containerized stack with:
- Web interface on port 8010
- REST API on port 8011
- FHIR API on port 8012
- MySQL database on port 3316
- Docker Compose orchestration
- Health check monitoring

## Use Cases

- **Clinic Management**: Run a complete medical practice
- **Telemedicine Platform**: Virtual consultations and remote care
- **Clinical Analytics**: Export data for population health analysis
- **Healthcare Integration**: Connect with labs, pharmacies, insurers
- **Medical Research**: Aggregate anonymized clinical data

## Integration Points

- **Messaging**: Twilio/Email for appointment reminders
- **Analytics**: Export to Postgres/Qdrant for analysis
- **Compliance**: Vault for secure credential storage
- **Monitoring**: Health metrics and usage statistics

## Security & Compliance

OpenEMR includes features for HIPAA compliance:
- Encrypted data transmission
- Audit logging
- Role-based access control
- Secure API authentication
- Patient consent tracking

## CLI Commands

```bash
# Lifecycle management
vrooli resource openemr manage install
vrooli resource openemr manage start [--wait]
vrooli resource openemr manage stop
vrooli resource openemr manage restart
vrooli resource openemr manage uninstall

# Content operations
vrooli resource openemr content add patient <options>
vrooli resource openemr content list patients [--limit N]
vrooli resource openemr content get patient <id>
vrooli resource openemr content remove patient <id>

# Appointment management
vrooli resource openemr content add appointment <options>
vrooli resource openemr content list appointments [--date DATE]
vrooli resource openemr content update appointment <id> <options>

# Data export
vrooli resource openemr content export <type> [--format fhir|csv|json]

# Status and logs
vrooli resource openemr status
vrooli resource openemr logs [--tail N]
vrooli resource openemr test [smoke|integration|all]
```

## Configuration

Key environment variables:
- `OPENEMR_PORT`: Web interface port (default: 8010)
- `OPENEMR_API_PORT`: REST API port (default: 8011)
- `OPENEMR_FHIR_PORT`: FHIR API port (default: 8012)
- `OPENEMR_DB_PORT`: MySQL port (default: 3316)
- `OPENEMR_DB_NAME`: Database name (default: openemr)
- `OPENEMR_ADMIN_USER`: Admin username
- `OPENEMR_ADMIN_PASS`: Admin password

## Troubleshooting

### Stack won't start
- Check Docker is running: `docker ps`
- Verify ports are available: `lsof -i :8010,8011,8012`
- Check logs: `vrooli resource openemr logs`
- Ensure Docker Compose is installed: `docker-compose --version`

### API authentication fails
- Ensure JWT token is valid
- Check API credentials in config
- Verify API service is running

### Database connection issues
- Confirm MySQL container is running
- Check database credentials
- Review connection logs

## Implementation Status

✅ **Implemented**:
- Docker Compose stack with OpenEMR 7.0.2
- MySQL 8.0 database with health checks
- CLI lifecycle commands (install, start, stop, restart)
- Health check endpoints
- Content management framework
- Port allocation in registry

⏳ **In Progress**:
- OAuth2 API authentication
- FHIR endpoint configuration
- Demo data seeding
- Appointment scheduling

## Resources

- [Official OpenEMR Site](https://www.open-emr.org)
- [OpenEMR GitHub](https://github.com/openemr/openemr)
- [API Documentation](https://www.open-emr.org/wiki/index.php/OpenEMR_REST_API)
- [FHIR Documentation](https://www.open-emr.org/wiki/index.php/OpenEMR_FHIR_API)