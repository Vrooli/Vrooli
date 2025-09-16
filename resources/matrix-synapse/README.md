# Matrix Synapse Resource

Matrix Synapse homeserver providing federated, encrypted, real-time communication infrastructure for Vrooli scenarios.

## Status: ✅ Fully Functional

All P0 requirements implemented and tested. Ready for production use.

## Quick Start

```bash
# Install and start Matrix Synapse
vrooli resource matrix-synapse manage install
vrooli resource matrix-synapse manage start

# Check health
vrooli resource matrix-synapse status

# Create a user
vrooli resource matrix-synapse content add-user alice password123

# Create a room
vrooli resource matrix-synapse content create-room "Team Chat"

# Send a message (requires room ID from create-room output)
vrooli resource matrix-synapse content send-message "!roomid:vrooli.local" "Hello team!"

# View logs
vrooli resource matrix-synapse logs
```

## Features

- ✅ **User Management**: Create and authenticate users with shared secret
- ✅ **Room Management**: Create, join, and manage collaboration rooms
- ✅ **Message Sending**: Send and receive messages via REST API
- ✅ **Federation Ready**: Well-known files configured for federation
- ✅ **PostgreSQL Backend**: Production-ready database storage
- ✅ **Health Monitoring**: Built-in health checks and status reporting
- ✅ **v2.0 Contract**: Full compliance with universal resource contract

## Configuration

Matrix Synapse uses environment variables for configuration:

```bash
# Required
MATRIX_SYNAPSE_PORT=8008               # API port
MATRIX_SYNAPSE_SERVER_NAME=vrooli.local # Your server name
SYNAPSE_DB_PASSWORD=secure_password     # PostgreSQL password
SYNAPSE_REGISTRATION_SECRET=secret      # User registration secret

# Optional
MATRIX_SYNAPSE_FEDERATION_ENABLED=true  # Enable federation
MATRIX_SYNAPSE_MAX_UPLOAD_SIZE=50M      # Max file upload size
```

## Usage Examples

### Creating Users

```bash
# Create a regular user
vrooli resource matrix-synapse content add-user alice

# Create a bot user
vrooli resource matrix-synapse content add-bot notification-bot

# List all users
vrooli resource matrix-synapse content list-users
```

### Managing Rooms

```bash
# Create a room
vrooli resource matrix-synapse content create-room "Team Chat"

# Send a message
vrooli resource matrix-synapse content send-message "#team-chat:vrooli.local" "Hello team!"
```

### API Access

The Matrix Client-Server API is available at `http://localhost:8008`:

```bash
# Check server version (health check)
curl http://localhost:8008/_matrix/client/versions

# Login (get access token)
curl -X POST http://localhost:8008/_matrix/client/v3/login \
  -d '{"type":"m.login.password","user":"alice","password":"password"}'

# Send a message
curl -X PUT http://localhost:8008/_matrix/client/v3/rooms/!roomid:vrooli.local/send/m.room.message/1 \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{"msgtype":"m.text","body":"Hello from API!"}'
```

## Integration with Scenarios

Matrix Synapse integrates seamlessly with Vrooli scenarios:

```javascript
// Example: Send notification from scenario
const matrix = await vrooli.resource('matrix-synapse');
await matrix.sendMessage('#alerts:vrooli.local', 'System alert: CPU usage high');

// Example: Create support room
const roomId = await matrix.createRoom('Support-Ticket-123');
await matrix.inviteUser(roomId, '@support-agent:vrooli.local');
```

## Testing

```bash
# Run smoke tests (basic health check)
vrooli resource matrix-synapse test smoke

# Run integration tests (full functionality)
vrooli resource matrix-synapse test integration

# Run all tests
vrooli resource matrix-synapse test all
```

## Troubleshooting

### Common Issues

1. **Port already in use**: Check if port 8008 is available
   ```bash
   lsof -i :8008
   ```

2. **Database connection failed**: Ensure PostgreSQL is running
   ```bash
   vrooli resource postgres status
   ```

3. **Registration disabled**: Check registration_shared_secret is set
   ```bash
   grep registration_shared_secret data/homeserver.yaml
   ```

### Debug Commands

```bash
# View detailed logs
vrooli resource matrix-synapse logs --tail 100

# Check configuration
vrooli resource matrix-synapse info --json

# Test database connection
vrooli resource matrix-synapse test integration --verbose
```

## Federation Setup

To enable federation with other Matrix servers:

1. Configure DNS SRV records or .well-known files
2. Ensure port 8448 is accessible (federation port)
3. Use a valid domain name (not .local for public federation)
4. Test federation at https://federationtester.matrix.org/

## Security Notes

- Always use PostgreSQL in production (not SQLite)
- Set strong registration_shared_secret
- Enable rate limiting to prevent abuse
- Use TLS for federation connections
- Regularly update Synapse version

## Resources

- [Matrix Specification](https://spec.matrix.org/)
- [Synapse Admin API](https://matrix-org.github.io/synapse/latest/admin_api/)
- [Element Web Client](https://app.element.io/)
- [Matrix Federation Tester](https://federationtester.matrix.org/)