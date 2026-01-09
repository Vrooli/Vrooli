/**
 * Privacy and Data documentation content
 *
 * Covers:
 * - How user data is handled
 * - What's stored locally vs sent to servers
 * - AI data usage
 * - Data deletion options
 */

export const PRIVACY_DATA_CONTENT = `
# Privacy & Data

Your privacy matters. This page explains how Vrooli Ascension handles your data, what's stored where, and how to manage or delete your information.

---

## Our Commitment

**We will never sell your data.** Period.

Vrooli Ascension is designed with privacy in mind:
- Your workflows, recordings, and browser data stay on your machine
- We don't track your browsing activity or collect analytics without consent
- Data is only sent to external services when explicitly required for functionality

---

## What's Stored Locally

The following data lives entirely on your device and is never transmitted:

### Browser Data

| Data | Location | Purpose |
| --- | --- | --- |
| **Settings** | Browser localStorage | Your preferences and configurations |
| **Session Profiles** | Browser storage | Saved browser states (cookies, localStorage) |
| **Custom Presets** | Browser localStorage | Your saved replay style presets |

### Project Data

| Data | Location | Purpose |
| --- | --- | --- |
| **Projects** | Local file system | Project folders with workflows |
| **Workflows** | JSON files on disk | Your automation definitions |
| **Recordings** | Local storage | Captured browser actions |

### Execution Data

| Data | Location | Purpose |
| --- | --- | --- |
| **Screenshots** | Temporary/local | Captured during execution |
| **Execution logs** | In-memory/local | Step-by-step results |
| **Replay frames** | Local cache | Data for replay viewing |

---

## What May Be Sent Externally

Some features require sending data to external services. Here's a complete list:

### AI-Assisted Navigation

When using AI features, the following may be sent to AI providers:

| Data Sent | Purpose | Provider |
| --- | --- | --- |
| **Page screenshots** | Visual context for AI navigation | Your configured AI provider |
| **Page text/DOM** | Understanding page structure | Your configured AI provider |
| **Your prompts** | Instructions for the AI | Your configured AI provider |

**Important:**
- AI providers are selected by you (OpenAI, Anthropic, etc.)
- Data is sent only when you actively use AI features
- We don't store or log AI conversations on our servers
- Review your AI provider's privacy policy for their data handling

### Video Export Rendering

When exporting MP4 or GIF files:

| Data Sent | Purpose |
| --- | --- |
| **Replay frames** | Server-side video rendering |
| **Style settings** | Apply your visual customizations |
| **Branding assets** | Include watermarks and intro/outro |

**Important:**
- Rendering happens on our servers for quality and speed
- Data is deleted immediately after rendering completes
- JSON exports are created locally and never leave your device

### Optional Analytics (If Enabled)

If you opt into analytics, we may collect:
- Feature usage statistics (which features are used)
- Error reports for debugging
- Performance metrics

**You can disable analytics at any time in Settings.**

---

## What We Never Collect

We explicitly do not collect or have access to:

- Passwords or authentication credentials
- Financial information or payment details
- Personal browsing history
- Content of pages you visit
- Your session cookies or localStorage data
- Workflow execution details (unless you export to our servers)

---

## AI Data Usage

### How AI Features Work

1. You enter a prompt describing what you want to do
2. A screenshot of the current page is captured
3. Both are sent to your configured AI provider
4. The AI returns suggested actions
5. You can review and approve before execution

### Your AI Provider's Role

- We integrate with third-party AI services (OpenAI, Anthropic, etc.)
- Your API keys are stored locally on your device
- Data sent to AI providers is subject to their privacy policies
- We recommend reviewing your provider's data retention policies

### Minimizing AI Data Exposure

- Avoid AI features on pages with sensitive information
- Use AI only when needed; recording mode sends no external data
- Clear your AI conversation history regularly
- Consider using AI providers with strong privacy commitments

---

## Managing Your Data

### Viewing Stored Data

- **Settings > Sessions** - View and manage session storage (cookies, localStorage)
- **Settings > Branding** - View uploaded branding assets
- **File system** - Browse project folders directly

### Deleting Data

Navigate to **Settings > Data** for deletion options:

#### Delete All Projects
- Removes all projects and their workflows
- Permanent and cannot be undone
- Does not affect sessions or settings

#### Delete All Exports
- Removes all exported recordings
- Frees up storage space
- Does not affect source workflows

#### Reset All Settings
- Restores all settings to defaults
- Clears custom presets and preferences
- Does not affect projects or exports

#### Delete Everything
- Combines all above operations
- Complete data wipe
- Returns application to fresh state

### Session Storage Management

Within each session profile:
- View all cookies by domain
- View all localStorage items by origin
- Delete individual items or clear all
- See storage size and item counts

---

## Data Security

### Local Storage Security

- Data is stored in browser sandboxed storage
- Protected by your operating system's user permissions
- Not accessible to other applications or websites

### API Key Security

- AI provider keys are stored locally
- Never transmitted to our servers
- Encrypted by browser storage mechanisms

### Network Security

- All external communication uses HTTPS
- No data is sent over unencrypted connections
- Server connections are authenticated

---

## Your Rights

You have complete control over your data:

- **Access** - All your data is visible and accessible on your device
- **Export** - Workflows are JSON files you can copy anywhere
- **Delete** - Remove any or all data at any time
- **Portability** - Your workflows work with standard Playwright

---

## Questions?

If you have questions about privacy or data handling:

- Check the **About** section for contact information
- Review our full privacy policy on our website
- Reach out via the feedback channels

We're committed to transparency and will gladly explain any aspect of our data practices.
`;
