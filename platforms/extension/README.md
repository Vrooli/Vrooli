# Browser Extension Template

## 🎯 Purpose

This is a **template** for adding browser extension capabilities to any application. It provides a starting point that AI agents can adapt to transform standalone apps into browser extensions.

## 🤖 Agent Instructions

When using this template to add browser extension capabilities to an app:

### 1. Understand the Target App
- Identify the app's main functionality
- Determine which features should be accessible from the browser toolbar
- Identify any APIs the extension needs to communicate with
- Check if the app needs to interact with web pages (content scripts)

### 2. Customization Points

Replace these placeholders throughout the template:

| Placeholder | Description | Example |
|------------|-------------|----------|
| `{{APP_NAME}}` | The application name | "Lead Generator Pro" |
| `{{APP_DOMAIN}}` | The app's primary domain | "leadgen.example.com" |
| `{{API_ENDPOINT}}` | The app's API base URL | "https://api.leadgen.example.com" |
| `{{APP_DESCRIPTION}}` | Brief description for manifest | "Automate lead generation from any webpage" |
| `{{PRIMARY_ACTION}}` | Main action the extension performs | "Extract contact information" |
| `{{ICON_SET}}` | Path to app's icon files | Custom icons or use defaults |

### 3. Core Components to Adapt

#### manifest.json
- Update permissions based on app requirements
- Add host_permissions for domains the app needs to access
- Configure commands for keyboard shortcuts
- Set appropriate content_scripts if needed

#### background.ts (Service Worker)
- Handles long-running tasks
- Manages communication between extension parts
- Interfaces with external APIs
- Common patterns included:
  - Message passing
  - Tab management
  - Storage sync
  - Context menus
  - Alarms for periodic tasks

#### index.tsx (Popup UI)
- The UI shown when clicking the extension icon
- Can be a mini version of the main app
- Or a control panel for the app's features
- Includes examples of:
  - State management
  - Chrome storage API usage
  - Message passing to background script

#### content.ts (Optional - Page Interaction)
- Runs in the context of web pages
- Can extract data, modify DOM, inject UI
- Template includes common patterns for:
  - Data extraction
  - Element highlighting
  - Overlay injection
  - Message passing with background

### 4. Build System Notes

- Uses Vite for fast development and building
- Supports React for popup UI (can be replaced with vanilla JS or other frameworks)
- TypeScript for type safety (optional, can use JavaScript)
- Hot reload in development mode

### 5. Common Extension Patterns Included

```typescript
// Authentication with external service
const authenticateUser = async () => { /* ... */ }

// Storing user preferences
chrome.storage.sync.set({ preferences: {...} })

// Capturing page content
chrome.tabs.executeScript({ code: 'document.body.innerText' })

// Creating context menus
chrome.contextMenus.create({ title: "Process with {{APP_NAME}}" })

// Badge notifications
chrome.action.setBadgeText({ text: "5" })

// Tab monitoring
chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => { /* ... */ })
```

### 6. Security Considerations

- Start with minimal permissions, add as needed
- Use content_security_policy to prevent XSS
- Never inject user data directly into pages
- Validate all data from content scripts
- Use HTTPS for all API communications
- Store sensitive data in chrome.storage.session (memory only)

### 7. Testing the Extension

1. Build the extension: `pnpm build`
2. Load in Chrome: chrome://extensions/ → Developer mode → Load unpacked
3. Test all permissions work correctly
4. Verify API communication
5. Check cross-browser compatibility if needed

### 8. Distribution Preparation

- Chrome Web Store requires:
  - Detailed description
  - Screenshots (1280x800 or 640x400)
  - Privacy policy
  - $5 developer registration
  
- Firefox Add-ons requires:
  - Mozilla account
  - Signed extension for distribution
  - AMO review process

## 📁 File Structure

```
extension/
├── src/
│   ├── background.ts    # Service worker (persistent logic)
│   ├── content.ts       # Content script (page interaction) - OPTIONAL
│   ├── index.tsx        # Popup UI (React component)
│   └── types.ts         # TypeScript definitions
├── public/
│   ├── manifest.json    # Extension configuration
│   └── icons/           # Extension icons
├── index.html           # Popup HTML
├── package.json         # Dependencies and scripts
├── vite.config.ts       # Build configuration
└── tsconfig.json        # TypeScript configuration
```

## 🚀 Quick Start for Agents

```bash
# 1. Copy this template to your app directory
cp -r platforms/extension /path/to/app/extension

# 2. Update package.json with app details
# 3. Replace placeholders in all files
# 4. Customize functionality based on app needs
# 5. Install dependencies
cd /path/to/app/extension && pnpm install

# 6. Build and test
pnpm build
```

## 💡 Example Transformations

### Lead Generation App → Extension
- Popup: Quick form to tag leads
- Content script: Highlight emails/phones on page
- Background: Sync leads to CRM API
- Context menu: "Save as lead" option

### Task Manager App → Extension  
- Popup: Today's tasks view
- Content script: Create tasks from selected text
- Background: Sync with task database
- Notifications: Task reminders

### Analytics App → Extension
- Popup: Current page analytics
- Content script: Track user interactions
- Background: Send data to analytics service
- Storage: Cache analytics locally

## 🛠️ Advanced Patterns

### Multi-page App in Extension
If the app is complex, consider:
- Using React Router in popup
- Opening app in new tab for full features
- Keeping only quick actions in popup

### Native Messaging
For apps that need system access:
- Set up native host application
- Configure native messaging in manifest
- Handle bidirectional communication

### Cross-Extension Communication
If the app has multiple extensions:
- Use runtime.sendMessage with extension ID
- Set up shared storage patterns
- Coordinate through external API

---

**Remember**: This template is a starting point. The goal is to help AI agents quickly understand browser extension patterns and adapt them to any app's specific needs. Don't be constrained by the template - use it as inspiration and modify freely! 