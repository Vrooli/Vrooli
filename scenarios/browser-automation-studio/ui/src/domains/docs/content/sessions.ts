/**
 * Sessions documentation content
 *
 * Covers browser session management including:
 * - What sessions are and why they matter
 * - Session profiles and storage state
 * - Fingerprint configuration
 * - Behavior simulation
 * - Browser signal settings (legacy anti_detection API fields)
 */

export const SESSIONS_CONTENT = `
# Browser Sessions

Sessions allow you to save and reuse browser state across workflow runs. This is essential for automating authenticated workflows or maintaining consistent browser configurations.

---

## What Are Sessions?

A **session** is a saved Playwright browser profile that includes:

- **Cookies** - Authentication tokens, session identifiers, preferences
- **LocalStorage** - Application data, user settings, cached information
- **Browser Configuration** - Viewport size, user agent, timezone, and more
- **Behavior Settings** - Typing, pointer, scroll, and pause timing

When you run a workflow with a session, it starts with all this saved state, so you don't need to log in again or reconfigure the browser.

---

## Managing Sessions

### Creating Sessions

1. Navigate to **Settings > Sessions**
2. Click **"Create New Session"**
3. Give it a descriptive name (e.g., "Production Login", "Test Account")

### Capturing Authentication

To save login state to a session:

1. Start a recording with your session selected
2. Navigate to the login page and sign in
3. The session automatically captures cookies and localStorage
4. An **"Auth saved"** badge appears when authentication state is stored

### Using Sessions in Workflows

- Select a session in the **Recording Header** before starting
- Workflows remember their default session
- Switch sessions any time to test with different accounts

---

## Session Configuration

### Fingerprint Settings

Fingerprint settings request browser identity and device characteristics for a
test environment. Support depends on the selected browser mode. Changing these
values does not establish compatibility with a site's checks.

#### Viewport
Set the browser window dimensions:
- **Width** - Screen width in pixels
- **Height** - Screen height in pixels

#### User Agent
The browser identification string sent to websites:
- **Presets** - Chrome Windows, Safari macOS, Firefox Linux, Mobile Chrome, Mobile Safari
- **Custom** - Enter any user agent string

#### Locale & Timezone
Regional settings that affect date/time display and language:
- **Locale** - Language code (e.g., \`en-US\`, \`fr-FR\`, \`ja-JP\`)
- **Timezone** - Time zone identifier (e.g., \`America/New_York\`, \`Europe/London\`)

#### Geolocation
Set test coordinates for location-based workflows:
- **Latitude** - GPS latitude (-90 to 90)
- **Longitude** - GPS longitude (-180 to 180)

#### Device Properties
Hardware characteristics reported to JavaScript:
- **Scale Factor** - Device pixel ratio (1x, 2x for Retina, etc.)
- **CPU Cores** - Reported \`navigator.hardwareConcurrency\`
- **Memory** - Reported \`navigator.deviceMemory\` in GB

---

## Interaction Timing

These settings change typing, pointer, scroll, and pause timing. They can make
recorded interactions less uniform, but they do not make a session human or
guarantee that a site will accept it.

### Typing Behavior

| Setting | Description |
| --- | --- |
| **Typing Speed** | Min/max delay between keystrokes (ms) |
| **Pre-Typing Delay** | Pause before starting to type (simulates reading) |
| **Paste Threshold** | Long text is pasted instead of typed (character limit) |
| **Typing Variance** | Character-aware delays (faster for common pairs, slower for capitals) |

### Mouse Movement

| Setting | Description |
| --- | --- |
| **Movement Style** | Linear (fast), Bezier (smooth curves), Natural (curved path) |
| **Jitter** | Random deviation in pixels (adds organic feel) |
| **Click Delay** | Min/max pause between clicks |

### Scroll Behavior

| Setting | Description |
| --- | --- |
| **Smooth** | Continuous scrolling animation |
| **Stepped** | Discrete scroll increments |

### Micro-Pauses

Random brief pauses during interactions:
- **Enabled** - Toggle random pauses on/off
- **Frequency** - How often pauses occur (percentage)

---

## Browser Signal Options

The legacy \`anti_detection\` profile options request changes to selected browser
signals. Their support and effect vary by browser mode and version. Sites can
use other browser, network, and behavior signals, so these options do not
guarantee access or a particular detection result. WebRTC changes can affect
site functionality and do not replace network-policy checks.

| Option | Requested change |
| --- | --- |
| **Disable Automation Flag** | Adjust a Chrome automation launch feature |
| **Patch navigator.webdriver** | Adjust the WebDriver property |
| **Patch navigator.plugins** | Adjust the reported plugin list |
| **Patch navigator.languages** | Adjust reported language values |
| **Patch WebGL Renderer** | Adjust reported WebGL renderer details |
| **Patch Canvas Fingerprint** | Adjust canvas output |
| **Headless Detection Bypass** | Legacy option name; no bypass guarantee |
| **Disable WebRTC** | Request WebRTC disablement; network behavior still needs separate validation |

### Presets

Quick-apply common configurations:
- **Default** - Standard Playwright settings
- **Stealth** - Legacy preset name for a group of signal and interaction settings; no detection guarantee
- **Mobile** - Mobile viewport and user agent
- **Custom** - Your saved configurations

---

## Storage Management

View and manage saved browser data within a session.

### Cookies

- View all cookies by domain
- See cookie values, expiration, and flags
- Delete individual cookies or clear all

### LocalStorage

- View localStorage items grouped by origin
- See keys and values
- Delete individual items or clear all

### Storage Stats

The session card shows:
- **Cookie count** - Number of saved cookies
- **Storage size** - Total data size in KB/MB

---

## Best Practices

### Use Separate Sessions For:

- **Different accounts** - One session per user account
- **Different environments** - Staging vs. production
- **Different test scenarios** - Clean state vs. pre-populated data

### Maintain Session Hygiene

- **Clear expired data** - Remove old cookies periodically
- **Rename descriptively** - "Admin Account - Staging" is better than "Session 1"
- **Delete unused sessions** - Keep your session list manageable

### When to Create a New Session

- Testing with a different user account
- Switching between environments
- Needing a clean slate without existing cookies/storage

---

## Troubleshooting

### "Auth saved" Badge Missing

- Ensure you completed the login flow within the recording
- Check that the site sets cookies (some use token-only auth)
- Try refreshing the session list

### Session Not Retaining Login

- Some sites expire sessions quickly
- Check cookie expiration times in session storage view
- The site may block or challenge this browser mode. No profile option guarantees acceptance; use only a supported mode and report the site-specific outcome.

### Workflows Fail After Session Change

- The site may have different content per account
- Selectors might need updating for different user states
- Verify the workflow steps match the session's expected state
`;
