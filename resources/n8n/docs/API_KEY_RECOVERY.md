# n8n API Key Recovery Guide

## ⚠️ Critical Information

**API keys can only be generated through the n8n UI**. There is no programmatic way to create API keys. If the n8n database is lost or reset, all API keys become invalid and must be manually recreated.

## Understanding API Key Persistence

### Storage Locations
- **API Key**: Stored in `.vrooli/secrets.json` (host filesystem)
- **n8n Database**: Stored in `~/.local/share/n8n/` (host filesystem)
- **User Accounts**: Stored within n8n database

### The Critical Relationship
```
API Key (in .vrooli/) ←→ User Account (in n8n database)
```
If the user account is lost or recreated, the API key becomes invalid even though it still exists in `.vrooli/secrets.json`.

## Operations That Invalidate API Keys

### High Risk Operations
1. **Uninstall with data removal** (`--action uninstall --remove-data yes`)
   - Completely wipes n8n database
   - All user accounts lost
   - All API keys become invalid

2. **Password reset** (`--action reset-password`)
   - Recreates container with new credentials
   - May invalidate existing API keys
   - User account might be recreated

3. **Recovery from corruption** (auto-recovery)
   - Wipes and recreates database
   - All user accounts lost
   - All API keys become invalid

4. **Database type change** (SQLite ↔ PostgreSQL)
   - Creates new database
   - All user accounts lost
   - All API keys become invalid

### Safe Operations
- `start`, `stop`, `restart` - Only affect container state
- `logs`, `status`, `info` - Read-only operations
- `execute`, `list-workflows` - Use existing configuration

## Manual Recovery Process

### Step 1: Verify API Key Status
```bash
# Check if your API key is still valid
./manage.sh --action test
```

### Step 2: Access n8n Web Interface
1. Navigate to `http://localhost:5678` (or your configured URL)
2. Login with your credentials:
   - Username: Usually `admin` (check container logs if unsure)
   - Password: Check container environment or logs

### Step 3: Create New API Key
1. Go to **Settings** (gear icon in sidebar)
2. Navigate to **n8n API** section
3. Click **Create API Key**
4. Give it a descriptive name (e.g., "Vrooli Integration")
5. **IMPORTANT**: Copy the API key immediately - it won't be shown again!

### Step 4: Save API Key to Vrooli
```bash
./manage.sh --action save-api-key --api-key "YOUR_NEW_API_KEY_HERE"
```

### Step 5: Verify New Key Works
```bash
./manage.sh --action test
```

## Emergency Recovery Scenarios

### Scenario 1: Lost Password
If you've lost the n8n login password but the container is running:
```bash
# Check current password in container
docker exec n8n-automation env | grep N8N_BASIC_AUTH_PASSWORD

# Or reset password (WARNING: may invalidate API keys)
./manage.sh --action reset-password
```

### Scenario 2: Database Corrupted
If the database is corrupted and auto-recovery has run:
1. All API keys are now invalid
2. Check backup directory: `~/.n8n-backup/`
3. If backup exists, you may be able to restore it
4. Otherwise, follow manual recovery process above

### Scenario 3: Container Deleted
If someone accidentally deleted the container:
```bash
# Check if data still exists
ls -la ~/.local/share/n8n/

# If data exists, just restart
./manage.sh --action start

# API keys should still work if database wasn't touched
```

## Prevention Best Practices

### 1. Regular Backups
Before any risky operation:
```bash
# Manual backup
cp -r ~/.local/share/n8n ~/.n8n-backup/manual-$(date +%Y%m%d-%H%M%S)
```

### 2. Document Your API Keys
Keep a secure record of:
- API key creation date
- Associated user account
- Purpose/integration name

### 3. Test Before Operations
Always test API key validity before and after operations:
```bash
# Before
./manage.sh --action test

# Perform operation
./manage.sh --action [operation]

# After
./manage.sh --action test
```

### 4. Use --yes Carefully
The `--yes` flag bypasses warnings. Always review warnings about API key risks before proceeding.

## Automated Safeguards

The n8n resource now includes:

1. **Pre-operation Warnings**: Warns before any operation that might invalidate API keys
2. **Post-operation Validation**: Automatically checks if API keys are still valid
3. **Automatic Backups**: Creates backups before destructive operations
4. **Clear Recovery Instructions**: Provides step-by-step recovery guidance when keys are invalidated

## Troubleshooting

### API Key Valid but Not Working
- Check n8n is running: `./manage.sh --action status`
- Verify network connectivity: `curl http://localhost:5678/healthz`
- Check container logs: `./manage.sh --action logs`

### Can't Access n8n UI
- Verify port isn't blocked: `ss -tulpn | grep 5678`
- Check Docker network: `docker network ls`
- Try direct container access: `docker exec -it n8n-automation sh`

### Multiple API Keys
n8n supports multiple API keys. If one is compromised:
1. Create a new key via UI
2. Update Vrooli configuration
3. Delete old key from n8n UI

## Support

For issues specific to:
- **n8n**: Check [n8n documentation](https://docs.n8n.io)
- **Vrooli Integration**: See main documentation or file an issue
- **API Key System**: This is an n8n limitation, not a Vrooli issue