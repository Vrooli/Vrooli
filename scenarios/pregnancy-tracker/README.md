# Pregnancy Tracker 🤰

Privacy-first pregnancy tracking with evidence-based information, multi-outcome support, and complete data sovereignty.

## Overview

Pregnancy Tracker is a comprehensive pregnancy monitoring application that prioritizes user privacy by storing all data locally with encryption. It provides week-by-week development tracking, symptom monitoring, appointment management, and evidence-based information with scientific citations - addressing the key frustrations women have with existing pregnancy apps.

## Key Features

- **Privacy-First**: All data encrypted and stored locally only
- **Evidence-Based**: Content includes scientific citations (addressing the 72% of apps that lack this)
- **Full-Text Search**: Find information quickly (only 24% of apps have this feature)
- **Multi-Outcome Support**: Handles all pregnancy outcomes sensitively
- **Multi-Tenant**: Safe household sharing with complete user isolation
- **Partner Access**: Controlled sharing with granular permissions
- **Medical Export**: Generate PDF/JSON reports for healthcare providers
- **Emergency Card**: Quick access to critical medical information
- **Comprehensive Tracking**: Symptoms, weight, appointments, kicks, contractions

## What Makes This Different

Based on extensive research of what women actually want:
- **Search functionality** - Most apps lack this basic feature
- **Scientific citations** - Evidence-based content, not just opinions
- **Flexible outcomes** - Not assuming every pregnancy ends in live birth
- **Safety information** - Clear guidance on when to seek medical help
- **Complete privacy** - No data mining, no employer access, no insurance snooping

## Quick Start

1. **Setup the scenario:**
   ```bash
   vrooli scenario setup pregnancy-tracker
   ```

2. **Start the services:**
   ```bash
   vrooli scenario run pregnancy-tracker
   ```

3. **Access the UI:**
   - Open http://localhost:37001 in your browser
   - Set your User ID for multi-tenant support
   - Enter your pregnancy details to begin tracking

## Integrations

### With Period Tracker
- Import menstrual history for accurate dating
- Seamless transition from TTC to pregnancy

### With Calendar
- Automatic appointment scheduling
- Due date and milestone reminders

### With Scenario Authenticator
- Multi-user household support
- Partner access management

## CLI Usage

```bash
# Check pregnancy status
pregnancy-tracker status

# Start tracking (with Last Menstrual Period date)
pregnancy-tracker start 2024-01-15

# Log daily symptoms
pregnancy-tracker log-daily

# Search for information
pregnancy-tracker search "morning sickness remedies"

# Get week-specific information
pregnancy-tracker week-info 20

# Export your data
pregnancy-tracker export json > my-pregnancy.json
pregnancy-tracker export pdf > pregnancy-report.pdf
```

## Privacy & Security

- **AES-256 Encryption**: All sensitive data encrypted at application layer
- **Local Storage Only**: No cloud sync, no external transmission
- **Multi-Tenant Safe**: Complete data isolation between users
- **Audit Logging**: Track all access for compliance
- **Zero Analytics**: No tracking, no telemetry, no data collection

## Development

### Architecture
```
pregnancy-tracker/
├── api/                 # Go API with encryption and search
├── cli/                 # Bash CLI for terminal access
├── ui/                  # JavaScript web interface
├── initialization/      # Database schema and seed data
│   └── postgres/        # Evidence-based content with citations
├── test/               # Validation suite
└── .vrooli/            # Service configuration
```

### API Development
```bash
cd api
go mod download
go build -o pregnancy-tracker-api .
./pregnancy-tracker-api
```

### UI Development
```bash
cd ui
npm install
npm start
```

## Data Model

- **Pregnancies**: Core tracking with gestational calculations
- **Daily Logs**: Encrypted symptoms, mood, measurements
- **Appointments**: Medical visits with reminders
- **Emergency Info**: Quick-access medical details
- **Search Content**: Evidence-based, cited information
- **Partner Access**: Granular permission control

## Testing

Run comprehensive validation:
```bash
vrooli scenario test pregnancy-tracker
```

## Roadmap

### Current (v1.0)
- ✅ Core pregnancy tracking
- ✅ Multi-tenant support
- ✅ Full-text search
- ✅ Evidence-based content
- ✅ Multiple outcome support

### Planned (v2.0)
- [ ] 3D fetal visualizations
- [ ] Voice notes via Whisper
- [ ] Wearable device integration
- [ ] Anonymous community features
- [ ] AI Q&A with stronger medical disclaimers

## Support

For issues or questions:
- Check the PRD.md for detailed requirements
- Run `vrooli scenario test pregnancy-tracker` (or `make test`) to execute the phased test suite
- Ensure all required resources are running

## License

Part of the Vrooli ecosystem - expanding human capability through privacy-respecting AI.

---

**Medical Disclaimer**: This app provides information but is not medical advice. Always consult your healthcare provider for medical decisions.

**Privacy Promise**: Your pregnancy data never leaves your device. You have complete control.
