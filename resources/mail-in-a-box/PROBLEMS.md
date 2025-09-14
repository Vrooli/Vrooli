# Mail-in-a-Box Resource Problems

## Architectural Mismatch

### Problem
The resource is named "mail-in-a-box" but uses `mailserver/docker-mailserver` instead of the actual Mail-in-a-Box project. This creates feature gaps:

**Expected (Mail-in-a-Box)**:
- Admin web panel on port 8543
- Roundcube webmail interface 
- Nextcloud for files/calendar/contacts
- DNS management
- Automated backups
- Web-based user management

**Actual (docker-mailserver)**:
- No admin web panel
- No webmail interface
- No calendar/contacts
- CLI-only management
- Basic email server only

### Impact
- P0 requirement "Webmail Access" cannot be fulfilled
- P1 requirements for Calendar/Contacts not possible
- No web UI for management tasks
- Limited scenario value for end-user applications

### Potential Solutions

1. **Switch to actual Mail-in-a-Box**
   - Use `mailinabox/mailinabox` Docker image
   - Provides all expected features
   - More complex setup requirements
   
2. **Add complementary containers**
   - Add Roundcube container for webmail
   - Add admin panel (PostfixAdmin or similar)
   - Add CalDAV/CardDAV server
   - Increases complexity but provides features

3. **Rename and reposition resource**
   - Rename to "docker-mailserver" 
   - Update PRD to match actual capabilities
   - Position as lightweight email server only

### Recommendation
For maximum value to Vrooli scenarios, option 1 (actual Mail-in-a-Box) or option 2 (docker-mailserver + webmail + admin) would be best. This would enable scenarios like customer support systems, marketing automation, and user communication features.

## Current Limitations

### No Web Interfaces
- Management requires CLI or direct container commands
- No user-friendly interface for non-technical users
- Scenarios cannot embed email management in their UIs

### Limited Integration Points
- No REST API for email management
- Cannot easily integrate with web scenarios
- Requires exec into container for most operations

### Missing Enterprise Features
- No multi-domain management UI
- No quota management
- No spam quarantine interface
- No backup management UI