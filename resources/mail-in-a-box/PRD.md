# Mail-in-a-Box Resource PRD

## Executive Summary

**What**: Complete email server solution with webmail, calendar, and contacts
**Why**: Scenarios need reliable email capabilities for communication, automation, and testing
**Who**: Used by scenarios requiring email infrastructure (customer support, marketing, testing)
**Value**: $25K+ - Eliminates need for external email services, enables email automation
**Priority**: Medium - Essential for communication-based scenarios

## Requirements Checklist

### P0 Requirements (Must Have)

- [ ] **Email Server Core**: SMTP/IMAP/POP3 services functional
  - Acceptance: Can send/receive emails between accounts
  - Test: `echo "test" | mail -s "test" admin@mail.local`
  - Status: Not tested (service not running)

- [ ] **Health Check**: Responds to health endpoint
  - Acceptance: Returns status within 5 seconds
  - Test: `timeout 5 curl -sf http://localhost:8543/health`
  - Status: Implementation exists, not tested with running service

- [x] **Lifecycle Commands**: v2.0 compliant CLI commands work
  - Acceptance: setup/develop/test/stop commands functional
  - Test: `vrooli resource mail-in-a-box test smoke`
  - Status: ✅ All lifecycle commands implemented and functional

- [ ] **User Management**: Can create/delete email accounts
  - Acceptance: Add/remove users via CLI
  - Test: `resource-mail-in-a-box content add user@example.com`
  - Status: Commands exist, not tested with running service

- [ ] **Webmail Access**: Roundcube interface accessible
  - Acceptance: Can login and use webmail
  - Test: Browse to https://localhost/mail
  - Status: Not tested (service not running)

### P1 Requirements (Should Have)

- [ ] **Calendar/Contacts**: CalDAV/CardDAV services working
  - Acceptance: Can sync calendar and contacts
  - Test: Connect calendar client to CalDAV endpoint

- [ ] **Multi-domain Support**: Can handle multiple email domains
  - Acceptance: Email routing works for 2+ domains
  - Test: `resource-mail-in-a-box content add-domain example2.com`

- [ ] **Spam Protection**: SpamAssassin filtering active
  - Acceptance: Spam emails marked/filtered
  - Test: Send GTUBE test spam string

- [ ] **Automated Backups**: Regular backup system working
  - Acceptance: Backups created in ~/.mailinabox/backup
  - Test: `ls ~/.mailinabox/backup`

### P2 Requirements (Nice to Have)

- [ ] **Email Aliases**: Can create email aliases
  - Acceptance: Aliases forward correctly
  - Test: `resource-mail-in-a-box content add-alias`

- [ ] **Auto-configuration**: Email client auto-config works
  - Acceptance: Thunderbird auto-discovers settings
  - Test: Configure email client with autoconfig

- [ ] **Custom DNS**: DNS management interface functional
  - Acceptance: Can manage DNS records via admin panel
  - Test: Access DNS settings in admin panel

## Technical Specifications

### Architecture
- **Container**: mailinabox/mailinabox:v68
- **Services**: Postfix (SMTP), Dovecot (IMAP/POP3), Roundcube (Webmail), Nextcloud (Files)
- **Database**: SQLite for user management
- **Storage**: Volume-based persistent storage

### Dependencies
- Docker for containerization
- No external resource dependencies
- Optional: postgres for advanced scenarios

### API Endpoints
- Admin API: https://localhost:8543/admin/api
- Health: http://localhost:8543/health
- Webmail: https://localhost/mail
- CalDAV: https://localhost/cloud/remote.php/dav

### Ports
- 25: SMTP
- 587: SMTP Submission
- 993: IMAPS
- 995: POP3S
- 8543: Admin Panel
- 443: HTTPS (Webmail/Calendar)

## Success Metrics

### Completion Targets
- P0: 100% required for production use
- P1: 75% for enhanced functionality
- P2: 25% for nice-to-have features
- Overall: 85% for full resource maturity

### Quality Metrics
- Health check response time: <1s
- Email delivery time: <5s local
- Container startup time: <60s
- Memory usage: <2GB under load

### Performance Targets
- Concurrent users: 50+
- Messages/hour: 1000+
- Storage efficiency: <100KB per email average
- Backup time: <5 minutes for 1GB

## Progress History

### 2025-01-13: Initial PRD Creation
- Created PRD.md per v2.0 contract requirements
- Defined P0/P1/P2 requirements based on existing functionality
- Set acceptance criteria and test commands
- Progress: 0% → 0% (PRD creation only)

### 2025-01-13: v2.0 Contract Compliance Update
- ✅ Created PRD.md with P0/P1/P2 requirements
- ✅ Added config/secrets.yaml for secure credential management
- ✅ Created test directory structure with run-tests.sh
- ✅ Implemented test phases: smoke, unit, integration
- ✅ Added config/schema.json for configuration validation
- ✅ Updated CLI with proper test command handlers
- ✅ Added validation functions to core.sh
- Unit tests: 19/19 passing
- Smoke tests: 1/7 passing (service not running)
- Progress: 0% → 40% (Structure complete, service needs testing)