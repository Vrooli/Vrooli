# Silent Founder OS - Installation & Setup Guide

## Downloading Vrooli Ascension

### Where to Download

Download the installer for your operating system from [vrooli.com](https://vrooli.com). After subscribing, you'll have access to download links for:

- **Windows:** `.exe` installer
- **macOS:** `.dmg` disk image (Universal binary for Intel and Apple Silicon)
- **Linux:** `.AppImage` (portable) or `.deb` package

### Verifying Your Download

Each download includes a SHA-256 checksum. To verify:

**Linux/macOS:**
```bash
sha256sum Vrooli-Ascension-*.AppImage
# Compare output to checksum on download page
```

**Windows (PowerShell):**
```powershell
Get-FileHash .\Vrooli-Ascension-Setup.exe -Algorithm SHA256
```

---

## System Requirements

### Minimum Requirements

| Component | Requirement |
|-----------|-------------|
| OS | Windows 10+, macOS 11+, or Linux (glibc 2.28+) |
| RAM | 4 GB |
| Disk Space | 500 MB (plus space for recordings) |
| Display | 1280x720 minimum |

### Recommended

| Component | Recommendation |
|-----------|----------------|
| RAM | 8 GB or more |
| Disk Space | 2 GB+ (recordings accumulate) |
| Display | 1920x1080 or higher |

The app bundles its own Chromium browser, so no separate browser installation is required.

---

## First-Run Experience

### Installation

**Windows:**
1. Run the `.exe` installer
2. Follow the installation wizard
3. Launch from Start Menu or desktop shortcut

**macOS:**
1. Open the `.dmg` file
2. Drag Vrooli Ascension to Applications
3. On first launch, right-click → Open (to bypass Gatekeeper)

**Linux:**
```bash
# AppImage (portable)
chmod +x Vrooli-Ascension-*.AppImage
./Vrooli-Ascension-*.AppImage

# Or install .deb
sudo dpkg -i vrooli-ascension_*.deb
```

### First Launch

1. **License activation:** Enter your subscription email when prompted
2. **Initial setup:** The app creates local storage directories for workflows, recordings, and screenshots
3. **Welcome tour:** A brief walkthrough explains the main interface

### Interface Overview

The main window has three areas:

- **Browser pane:** A full Chromium browser where you work normally
- **Timeline panel:** Real-time capture of all browser actions
- **Sidebar:** Access to projects, workflows, and settings

---

## Creating and Running Workflows

### Recording a Workflow

1. **Start browsing:** Navigate to any website in the built-in browser
2. **Perform actions:** Click, type, scroll, navigate—everything is captured
3. **Review timeline:** See each action in the timeline panel
4. **Select range:** Click the start point, shift-click the end point
5. **Create workflow:** Click "Save as Workflow" and name it

### Running Workflows

**On demand:**
1. Open the workflow from the sidebar
2. Click "Run" in the workflow header
3. Watch real-time execution with screenshots

**On schedule:**
1. Open workflow settings
2. Enable "Schedule" and set frequency (hourly, daily, weekly, cron)
3. The workflow runs automatically at specified times

**Via CLI:**
```bash
browser-automation-studio workflow execute "my-workflow" --wait
```

**Via API:**
```bash
curl -X POST http://localhost:${API_PORT}/api/v1/workflows/execute \
  -H "Content-Type: application/json" \
  -d '{"workflow_id": "my-workflow"}'
```

---

## Editing Actions Retroactively

One of Vrooli Ascension's key features is retroactive editing—you can modify captured actions after the fact.

### Editing Individual Actions

1. Click any action in the timeline
2. Modify properties in the inspector panel:
   - **Selectors:** Change the target element
   - **Values:** Update typed text or selected options
   - **Timing:** Adjust wait durations
   - **Assertions:** Add checks for element state

### Adding New Actions

1. Position the cursor in the timeline where you want to insert
2. Click "+ Add Action" in the toolbar
3. Choose action type (click, type, navigate, wait, assert, etc.)
4. Configure the action properties

### Removing Actions

1. Select one or more actions in the timeline
2. Press Delete or click "Remove" in the toolbar
3. Confirm removal

### Reordering Actions

1. Drag actions in the timeline to reorder
2. The workflow updates automatically
3. Be mindful of dependencies (e.g., can't click before navigating)

---

## Scheduling Workflows

### Schedule Options

| Frequency | Description |
|-----------|-------------|
| Hourly | Every N hours |
| Daily | At a specific time each day |
| Weekly | On specific days at a set time |
| Cron | Custom cron expression for advanced scheduling |

### Setting Up a Schedule

1. Open the workflow
2. Click the "Schedule" icon in the toolbar
3. Choose frequency and configure timing
4. Enable the schedule
5. The workflow runs automatically (app must be open)

### Viewing Scheduled Runs

- Past executions appear in the "Executions" tab
- Each run includes status, duration, and screenshot evidence
- Failed runs show error details and the failing step

---

## Assertions and Logs

### Adding Assertions

Assertions verify that the page is in an expected state. Add them to any workflow step:

1. Select a step in the workflow
2. Click "Add Assertion"
3. Choose assertion type:
   - **Element exists:** Check that a selector is present
   - **Text contains:** Verify text content
   - **Attribute equals:** Check element attributes
4. Configure expected values
5. Set behavior on failure (stop workflow or continue)

### Viewing Logs

Each workflow execution captures:

- **Console logs:** JavaScript console output
- **Network requests:** URLs, status codes, response times
- **Screenshots:** Full-page capture at each step
- **Assertion results:** Pass/fail with details

Access logs from:
- **UI:** Click any execution in the history, then view the "Logs" tab
- **CLI:** `browser-automation-studio execution logs <execution-id>`

---

## Viewing and Exporting Replays

### Replay Viewer

1. Open any completed execution
2. Click the "Replay" tab
3. Watch the recorded session with:
   - Animated cursor movement
   - Click highlights
   - Scroll animations
   - Step-by-step navigation

### Styling Replays

Before exporting, customize the replay appearance:

- **Highlights:** Emphasize clicked elements
- **Masks:** Blur sensitive areas
- **Zoom:** Focus on specific regions
- **Theme:** Apply color schemes for branding

### Exporting

**HTML Package:**
```bash
browser-automation-studio execution render <execution-id> --output ./replay-folder
```
Creates a self-contained HTML file with embedded assets.

**Video (MP4/WEBM):**
```bash
browser-automation-studio execution render-video <execution-id> --output ./demo.mp4
```
Generates a video file suitable for product demos or marketing.

**JSON Metadata:**
```bash
browser-automation-studio execution export <execution-id> --output ./export.json
```
Exports structured data for custom processing or integration.

---

## Quick Reference

| Task | How |
|------|-----|
| Create workflow | Browse → Select timeline range → Save as Workflow |
| Run workflow | Open workflow → Click Run |
| Schedule workflow | Workflow settings → Enable Schedule |
| Add assertion | Select step → Add Assertion → Configure |
| View execution logs | Executions tab → Select run → Logs tab |
| Export replay video | CLI: `execution render-video <id>` |
| Get help | Sidebar → Documentation, or [github.com/Vrooli/Vrooli/tree/master/scenarios/browser-automation-studio/docs](https://github.com/Vrooli/Vrooli/tree/master/scenarios/browser-automation-studio/docs) |
