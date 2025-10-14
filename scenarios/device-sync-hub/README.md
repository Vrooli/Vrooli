# Device Sync Hub

Cross-device file and clipboard synchronization for seamless data sharing between phones, laptops, tablets, and other devices.

## 🎯 Purpose

Device Sync Hub solves the daily friction of moving files and text between devices. Simply upload content from one device and instantly access it from any other device on your network. Perfect for:

- Sharing photos from phone to desktop
- Syncing clipboard text between devices  
- Moving files without email or cloud storage
- Quick note sharing across devices
- Voice memo to desktop workflows

## ✨ Key Features

- **Real-time Sync**: WebSocket-based instant synchronization across all connected devices
- **Any File Type**: Upload and share any file type with configurable size limits
- **Mobile-First**: Responsive UI optimized for phone usage with desktop support
- **Clipboard Integration**: Share clipboard content directly between devices
- **Automatic Cleanup**: Files expire automatically (configurable, default 24 hours)
- **Secure Access**: Integration with scenario-authenticator for user authentication
- **Thumbnail Preview**: Automatic thumbnail generation for images
- **CLI Support**: Full command-line interface for scripting and automation

## 🚀 Quick Start

### 1. Setup and Run

```bash
# Start the Device Sync Hub scenario
vrooli scenario start device-sync-hub
# Or using Makefile:
cd scenarios/device-sync-hub && make start

# The lifecycle system will display access URLs with dynamically allocated ports
# Example output:
# - API: http://localhost:17402
# - Web UI: http://localhost:37155
# - CLI: device-sync-hub --help

# Check current ports and status:
vrooli scenario status device-sync-hub
```

### 2. Authentication Setup

Device Sync Hub requires scenario-authenticator for secure access:

```bash
# Make sure scenario-authenticator is running first
vrooli scenario run scenario-authenticator

# Create your first user account
scenario-authenticator user create user@example.com MyPassword123! user
```

### 3. Web Interface

Visit the UI URL displayed in the status output (typically `http://localhost:37155`) on any device and:

1. Login with your credentials
2. Upload files via drag-and-drop or tap to select
3. Share text content or clipboard data
4. See real-time sync across all your connected devices

### 4. CLI Usage

```bash
# Configure CLI environment (use ports from status output)
export API_URL=http://localhost:17402
export AUTH_URL=http://localhost:15785

# Or use with environment variables directly:
API_URL=http://localhost:17402 AUTH_URL=http://localhost:15785 device-sync-hub status

# Upload a file
device-sync-hub upload photo.jpg

# Share some text
device-sync-hub upload-text "Remember to call Mom"

# Share clipboard content (macOS example)
device-sync-hub upload-text "$(pbpaste)" --clipboard

# List all synced items
device-sync-hub list

# Filter items by type
device-sync-hub list --filter file

# Search for specific content
device-sync-hub list --search "meeting notes"

# Download an item
device-sync-hub download a1b2c3d4-e5f6-7890-abcd-ef1234567890

# Check status
device-sync-hub status --verbose
```

## 📱 Mobile Setup

### For Remote Access (Optional)

To access from devices outside your local network:

1. **Setup secure tunnel** (you handle this part)
   - Use ngrok, Cloudflare Tunnel, or similar
   - Point to dynamic ports from `vrooli scenario status device-sync-hub` (e.g. UI: 37155, API: 17402)

2. **Access from mobile device**
   - Visit your tunnel URL in mobile browser
   - Login with your credentials
   - Add to home screen for app-like experience

3. **Mobile Usage Tips**
   - UI is optimized for touch interaction
   - Large touch targets (44px minimum)
   - Drag-and-drop works on mobile browsers
   - Responsive design adapts to all screen sizes

## 🏗️ Technical Architecture

### Services

- **API Server**: Go-based API with WebSocket support (dynamic port allocation)
- **Web UI**: Mobile-first responsive interface (dynamic port allocation)
- **CLI Tool**: Full-featured command-line interface
- **Database**: PostgreSQL for metadata and user sessions
- **Authentication**: scenario-authenticator integration
- **Storage**: Local filesystem with automatic cleanup

### Dependencies

- **Required**: scenario-authenticator, postgres
- **Optional**: redis (for enhanced caching and WebSocket management)

### Data Flow

```
Mobile Device ←→ WebSocket ←→ API Server ←→ Database
     ↓                           ↓
Desktop Browser ←→ WebSocket ←→ File Storage
     ↓
CLI Tool ←→ API Server
```

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `API_PORT` | Dynamic | API server port (allocated by lifecycle system) |
| `UI_PORT` | Dynamic | Web UI port (allocated by lifecycle system) |
| `AUTH_SERVICE_URL` | http://localhost:15785 | Authentication service URL |
| `STORAGE_PATH` | ./data/files | File storage directory |
| `MAX_FILE_SIZE` | 10485760 | Maximum file size (10MB) |
| `DEFAULT_EXPIRY_HOURS` | 24 | Default item expiration time |

**Note**: Ports are dynamically allocated. Use `vrooli scenario status device-sync-hub` to see current ports.

### Settings

**Note**: Settings are read-only and configured via environment variables for security.
- **File Size Limit**: 1MB - 100MB (default: 10MB) - Set via `MAX_FILE_SIZE`
- **Expiration Time**: 1 hour - 1 week (default: 24 hours) - Set via `DEFAULT_EXPIRY_HOURS`
- **Thumbnail Size**: Image thumbnail pixel size (default: 200px) - Set via `THUMBNAIL_SIZE`

## 💡 Usage Examples

### Cross-Device Workflows

**Photo Editing Workflow**:
1. Take photo on phone
2. Upload via mobile web interface
3. Access instantly on desktop for editing
4. Share edited result back

**Voice Memo to Text**:
1. Record voice memo on phone
2. Upload audio file via sync hub
3. Download on desktop for transcription
4. Share transcribed text back to phone

**Code Snippet Sharing**:
```bash
# On desktop - share code snippet
device-sync-hub upload-text "console.log('Hello World');" --clipboard

# On mobile - access the code for reference
device-sync-hub list --filter clipboard
```

**Bulk File Transfer**:
```bash
# Upload multiple files
for file in *.jpg; do
    device-sync-hub upload "$file"
done

# List and download on other device
device-sync-hub list --json | jq -r '.items[].id' | while read id; do
    device-sync-hub download "$id"
done
```

## 🔌 API Integration

Other scenarios can integrate with Device Sync Hub:

### Upload via API

```javascript
// Get API URL from scenario status
const apiUrl = 'http://localhost:17402'; // Use actual port from status
const response = await fetch(`${apiUrl}/api/v1/sync/upload`, {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${authToken}`
  },
  body: formData // or JSON for text content
});
```

### List Items with Filters

```javascript
// Get API URL from scenario status
const apiUrl = 'http://localhost:17402'; // Use actual port from status

// Filter by type
const files = await fetch(`${apiUrl}/api/v1/sync/items?type=file`, {
  headers: { 'Authorization': `Bearer ${authToken}` }
});

// Search for content
const results = await fetch(`${apiUrl}/api/v1/sync/items?search=meeting`, {
  headers: { 'Authorization': `Bearer ${authToken}` }
});

// Combine filters
const filtered = await fetch(`${apiUrl}/api/v1/sync/items?type=text&search=important`, {
  headers: { 'Authorization': `Bearer ${authToken}` }
});
```

### WebSocket Connection

```javascript
// Get API URL from scenario status
const apiUrl = 'ws://localhost:17402'; // Use actual port from status
const ws = new WebSocket(`${apiUrl}/api/v1/sync/websocket`);

ws.onopen = () => {
  ws.send(JSON.stringify({
    type: 'auth',
    token: authToken,
    device_info: { platform: 'web' }
  }));
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  if (message.type === 'item_added') {
    console.log('New item:', message.item);
  }
};
```

## 🧪 Testing

```bash
# Run all tests
vrooli scenario test device-sync-hub

# Test specific components
device-sync-hub status --json
# Use actual ports from status output
curl -f http://localhost:17402/health  # API health check
curl -f http://localhost:37155/  # UI accessibility
```

## 📊 Monitoring

### Health Checks

- API: `GET /health` - Service health status
- UI: `GET /` - Web interface availability
- CLI: `device-sync-hub status` - Comprehensive status

### Metrics Available

- Active sync items per user
- WebSocket connection count
- File upload/download rates
- Storage usage and cleanup efficiency
- Authentication success/failure rates

## 🔒 Security

- **Authentication**: Full integration with scenario-authenticator
- **Network Security**: Local network only unless you setup external tunnel
- **Data Encryption**: Files stored locally with automatic cleanup
- **Access Control**: User-based access with JWT token validation
- **Audit Trail**: All actions logged with user ID and timestamp

## 🔄 Integration with Other Scenarios

Device Sync Hub enables these future scenarios:

- **Mobile Agent Assistant**: Phone agent uploads data, desktop agent processes
- **Voice Processing Pipeline**: Voice memos → transcription → agent workflows
- **Research Assistant**: Mobile field notes sync to desktop research tools
- **Photo Analysis Workflows**: Mobile capture → desktop AI analysis
- **Multi-Device Collaboration**: Shared workspaces across devices

## ⚡ Performance

- **Upload Speed**: >5MB/s on local network
- **Sync Latency**: <100ms for text content
- **Concurrent Devices**: Supports 10+ devices simultaneously
- **Memory Usage**: <512MB with typical usage
- **Storage**: Automatic cleanup prevents unbounded growth

## 🆘 Troubleshooting

### Connection Issues

```bash
# Check service status
device-sync-hub status --verbose

# Test API connectivity (use actual port from status)
curl http://localhost:17402/health

# Check authentication
device-sync-hub login
```

### WebSocket Problems

1. Check firewall settings for dynamically allocated ports (see `vrooli scenario status`)
2. Verify WebSocket support in browser (all modern browsers support it)
3. Check browser console for connection errors

### Mobile Access Issues

1. Ensure mobile device is on same network
2. Use IP address instead of localhost: `http://192.168.1.100:37155` (use actual UI port)
3. Check HTTPS requirements (some features need secure context)

### File Upload Failures

1. Check file size against limit: `device-sync-hub settings`
2. Verify disk space: `df -h`
3. Check permissions on storage directory

## 📚 Related Scenarios

- **scenario-authenticator**: Required for user authentication
- **voice-to-text-processor**: Process uploaded voice memos
- **image-analysis-pipeline**: Analyze uploaded images
- **research-assistant**: Enhanced with cross-device note sync

## 🔮 Future Enhancements

- **Cloud Storage Integration**: Backup to S3/GCP for large files
- **Offline Support**: PWA with offline capability and sync when online
- **File Versioning**: Track changes to updated items  
- **Collaboration Features**: Share items with other users
- **Advanced Search**: Full-text search across all content
- **Mobile App**: Native iOS/Android apps for better integration

---

**This scenario represents a permanent capability addition to Vrooli - every cross-device workflow built on this foundation enhances the intelligence and connectivity of the entire platform.**