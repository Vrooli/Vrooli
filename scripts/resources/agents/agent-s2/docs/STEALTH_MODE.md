# Agent-S2 Stealth Mode Documentation

## Overview

Agent-S2's Stealth Mode provides advanced anti-detection capabilities to bypass bot detection systems on websites. It includes browser fingerprint randomization, session persistence, and various evasion techniques.

## Features

### 1. WebDriver Detection Bypass
- Removes `navigator.webdriver` flag
- Hides automation indicators
- Disables Marionette and other automation extensions

### 2. Browser Fingerprint Randomization
- **User Agent Rotation**: Randomizes user agents based on profile type (residential, mobile, datacenter)
- **Canvas Fingerprinting Defense**: Adds noise to canvas operations
- **WebGL Randomization**: Randomizes WebGL parameters and vendor strings
- **Audio Context Noise**: Adds variations to audio fingerprinting
- **Hardware Spoofing**: Randomizes CPU cores, memory, and other hardware indicators

### 3. Network & Privacy Protection
- **WebRTC Leak Prevention**: Disables WebRTC to prevent IP leaks
- **Timezone Spoofing**: Sets timezone to UTC or randomizes based on profile
- **Language/Locale Randomization**: Varies browser language settings
- **DNS Prefetch Disabled**: Prevents DNS-based tracking

### 4. Session Persistence
- **Cookie Storage**: Saves and restores cookies between sessions
- **LocalStorage/SessionStorage**: Persists web storage data
- **Auth Token Management**: Maintains authentication across sessions
- **Encryption**: AES-256 encryption for stored session data

### 5. Browser State Persistence
- **Window/Tab State**: Saves and restores browser windows and tabs
- **Navigation History**: Maintains browsing history
- **Scroll Positions**: Restores page scroll states

## Configuration

### Environment Variables

```bash
# Enable/disable stealth mode (default: true)
AGENT_S2_STEALTH_MODE_ENABLED=true

# Session storage path
AGENT_S2_SESSION_STORAGE_PATH=/home/agents2/.agent-s2/sessions

# Enable session data persistence (default: true)
AGENT_S2_SESSION_DATA_PERSISTENCE=true

# Enable session state persistence (default: false)
AGENT_S2_SESSION_STATE_PERSISTENCE=false

# Enable session encryption (default: true)
AGENT_S2_SESSION_ENCRYPTION=true

# Session TTL in days (default: 30)
AGENT_S2_SESSION_TTL_DAYS=30

# Stealth profile type (residential, mobile, datacenter)
AGENT_S2_STEALTH_PROFILE_TYPE=residential
```

## Usage

### Basic Commands

```bash
# Enable stealth mode
./manage.sh --action configure-stealth --stealth-enabled yes

# Test stealth effectiveness
./manage.sh --action test-stealth

# Test with custom URL
./manage.sh --action test-stealth --stealth-url https://fingerprint.com/demo

# List saved session profiles
./manage.sh --action list-sessions

# Reset session data for a profile
./manage.sh --action reset-session-data --profile myprofile

# Export session to file
./manage.sh --action export-session --profile myprofile --output session.json

# Import session from file
./manage.sh --action import-session --profile newprofile --input session.json
```

### API Endpoints

```python
# Configure stealth settings
POST /stealth/configure
{
    "enabled": true,
    "features": {
        "fingerprint_randomization": true,
        "webdriver_hiding": true,
        "user_agent_rotation": true
    }
}

# Get stealth status
GET /stealth/status

# Test stealth effectiveness
POST /stealth/test
{
    "url": "https://bot.sannysoft.com/"
}

# Create profile
POST /stealth/profile/create
{
    "profile_id": "shopping_profile",
    "profile_type": "residential"
}

# List profiles
GET /stealth/profile/list

# Activate profile
PUT /stealth/profile/{profile_id}/activate

# Save current session
POST /stealth/session/save

# Reset session
DELETE /stealth/session/reset
```

### Python Client Example

```python
import requests

# Base URL
base_url = "http://localhost:4113"

# Enable stealth mode
response = requests.post(f"{base_url}/stealth/configure", json={
    "enabled": True,
    "features": {
        "fingerprint_randomization": True,
        "session_persistence": True
    }
})

# Create a profile
response = requests.post(f"{base_url}/stealth/profile/create", json={
    "profile_id": "amazon_shopping",
    "profile_type": "residential"
})

# Use AI with stealth profile
response = requests.post(f"{base_url}/ai/command", json={
    "command": "Navigate to amazon.com and search for laptops",
    "options": {
        "stealth": {
            "enabled": True,
            "profile": "residential"
        },
        "session": {
            "profile_id": "amazon_shopping",
            "restore_data": True
        }
    }
})
```

## Profile Types

### Residential
- Mimics home users with typical hardware/software
- Includes common browser extensions
- Standard screen resolutions (1920x1080, 1366x768)
- 4-16 CPU cores, 8-32GB RAM

### Mobile
- Mobile user agents (iOS Safari, Android Chrome)
- Touch support enabled
- Mobile viewports (390x844, 412x915)
- 4-8 CPU cores, 4-8GB RAM

### Datacenter
- Server-like configurations
- No plugins or extensions
- High-spec hardware (16-64 cores, 32-128GB RAM)
- Generic user agents

## Testing Stealth Effectiveness

### Recommended Test Sites

1. **Bot.SannySOFT**: https://bot.sannysoft.com/
   - Tests basic bot detection signals
   - Shows WebDriver, Chrome, and automation flags

2. **BrowserLeaks**: https://browserleaks.com/
   - Comprehensive fingerprinting tests
   - WebRTC, Canvas, WebGL, Audio tests

3. **FingerprintJS Demo**: https://fingerprint.com/demo
   - Advanced fingerprinting detection
   - Shows confidence scores

4. **CreepJS**: https://abrahamjuliot.github.io/creepjs/
   - Deep browser fingerprinting analysis
   - Detects lies and inconsistencies

### Expected Results

With stealth mode enabled:
- ✅ WebDriver: Not detected
- ✅ Headless: Not detected  
- ✅ Automation: No indicators
- ✅ Fingerprint: Randomized each session
- ✅ Canvas/WebGL: Noise added
- ✅ Timezone: Consistent with profile

## Limitations

1. **Advanced Bot Detection**: Some sites use proprietary detection that may still identify automation
2. **Behavioral Analysis**: Mouse movements and timing patterns may still reveal automation
3. **Session Limits**: Stored sessions expire after configured TTL
4. **Performance Impact**: Stealth features add slight overhead to operations

## Best Practices

1. **Use Appropriate Profiles**: Match profile type to target site (residential for shopping, mobile for apps)
2. **Rotate Sessions**: Don't reuse the same session indefinitely
3. **Add Delays**: Include realistic delays between actions
4. **Vary Behavior**: Randomize navigation patterns
5. **Monitor Detection**: Regularly test against detection services

## Troubleshooting

### Sessions Not Persisting
- Check `SESSION_DATA_PERSISTENCE` is enabled
- Verify storage path has write permissions
- Ensure container has persistent volume

### Still Detected as Bot
- Try different profile types
- Add behavioral randomization
- Check for WebRTC leaks
- Verify all stealth features are enabled

### Performance Issues
- Disable unused stealth features
- Reduce fingerprint complexity
- Use datacenter profile for non-critical sites

## Security Considerations

1. **Session Data**: Contains sensitive cookies and auth tokens
2. **Encryption**: Always enable for production use
3. **Access Control**: Limit API access to trusted clients
4. **Audit Logs**: Monitor session usage
5. **Compliance**: Respect website ToS and robots.txt