/**
 * Replay and Export documentation content
 *
 * Covers:
 * - Replay presentation modes
 * - Export formats (MP4, GIF, JSON)
 * - Customization options
 * - Branding and watermarks
 */

export const REPLAY_EXPORT_CONTENT = `
# Replay & Export

Vrooli Ascension lets you replay workflow executions and export them as videos, GIFs, or data packages. Customize the presentation with branding, animations, and visual effects.

---

## Replay Overview

After a workflow executes, you can replay the results to:

- **Review what happened** - Step through each action with screenshots
- **Debug failures** - See exactly where and why a step failed
- **Create demos** - Export polished videos for sharing

### Starting a Replay

1. Execute a workflow or select a past execution
2. The replay appears in the preview panel
3. Use playback controls to navigate through frames

---

## Export Formats

### MP4 Video

**Best for:** Sharing, presentations, documentation

- Server-rendered at 1080p quality
- Uses your configured replay styling
- Smooth transitions between frames
- Includes cursor animations and click effects

### Animated GIF

**Best for:** Quick sharing, social media, chat

- Looped highlight reel
- Smaller file size than video
- Respects your replay styling
- Compatible everywhere images work

### JSON Package

**Best for:** Developers, tooling, archival

- Raw data bundle with all metadata
- Frame images and timing information
- Can be re-rendered with custom tooling
- Downloaded via file picker to chosen location

---

## Export Options

### Render Source

For video and GIF exports, choose the source:

| Option | Description |
| --- | --- |
| **Auto** | Prefer native recording, fall back to replay frames |
| **Recorded Video** | Use Playwright's native video capture (when available) |
| **Replay Frames** | Use stylized replay frames (best for cursor control) |

### Dimensions

| Preset | Resolution | Use Case |
| --- | --- | --- |
| **1080p** | 1920 x 1080 | Full HD presentations |
| **720p** | 1280 x 720 | Web sharing, smaller files |
| **Custom** | 320 - 3840px | Specific requirements |

### File Naming

- Set a custom filename (without extension)
- Extension is added automatically (.mp4, .gif, .json)
- Preview shows the final filename

---

## Presentation Modes

Choose how the browser window appears in replays:

### Browser Frame

Full browser window with chrome (address bar, buttons):
- Most realistic representation
- Good for software demos
- Shows the complete user experience

### Desktop

Browser on a desktop background:
- Includes optional device frame (MacBook, iPhone, etc.)
- More polished, marketing-ready look
- Customizable background

### Fullscreen

Just the webpage content:
- No browser chrome or framing
- Maximum focus on the content
- Clean, minimal appearance

---

## Visual Customization

### Quick Presets

Apply pre-configured styles instantly:

- **Professional** - Clean, corporate appearance
- **Casual** - Friendly, approachable style
- **Minimal** - Just the essentials
- **Dark Mode** - Dark background and chrome
- **Randomize** - Surprise yourself with random settings

Create and save your own custom presets for quick reuse.

### Browser Chrome

| Setting | Options |
| --- | --- |
| **Appearance** | Light, Dark, Tinted |
| **Scale** | Size of browser window in desktop mode |

### Background

| Type | Description |
| --- | --- |
| **Solid Color** | Single color backdrop |
| **Gradient** | Two-color gradient |
| **Image** | Upload custom background |
| **Blur** | Blurred version of page content |

### Device Frames

Wrap the browser in a device mockup:
- MacBook Pro
- iPhone
- iPad
- Generic laptop/monitor
- None (just the browser)

---

## Cursor Customization

### Cursor Theme

| Theme | Description |
| --- | --- |
| **Default** | Standard arrow cursor |
| **Modern** | Sleek contemporary design |
| **Classic** | Traditional cursor style |
| **Pointer** | Hand pointer |
| **Circle** | Circular cursor |
| **Triangle** | Triangular cursor |

### Cursor Behavior

| Setting | Description |
| --- | --- |
| **Initial Position** | Where cursor starts (center, corner, etc.) |
| **Scale** | Size of the cursor |
| **Speed Profile** | Linear, Ease In, Ease Out, Ease In/Out, Instant |
| **Path Style** | Linear, Arc Up, Arc Down, Cubic, Organic |

### Click Animation

Visual feedback when clicking:
- **Ring** - Expanding circle
- **Pulse** - Pulsing dot
- **Dot** - Simple dot indicator

---

## Motion & Timing

### Frame Duration

How long each frame displays (0.8s to 6s):
- Shorter = faster, snappier feel
- Longer = more time to read and understand

### Playback Options

| Option | Description |
| --- | --- |
| **Auto-play** | Start replays automatically when opened |
| **Loop** | Restart automatically when finished |

---

## Branding

Add your identity to exports with branding elements.

### Watermark

Persistent logo overlay:
- **Position** - Corner placement
- **Size** - Logo dimensions
- **Opacity** - Transparency level
- **Margin** - Distance from edges

### Intro Card

Title slide before the replay starts:
- **Title** - Main heading text
- **Subtitle** - Secondary text
- **Logo** - Company/product logo
- **Background** - Color or image
- **Duration** - How long to display

### Outro Card

Closing slide with call-to-action:
- **CTA Text** - Call-to-action message
- **CTA URL** - Link destination
- **Logo** - Company/product logo
- **Background** - Color or image
- **Duration** - How long to display

---

## Asset Management

Manage uploaded images for branding:

### Uploading Assets

1. Go to **Settings > Branding**
2. Click to upload logos or backgrounds
3. Images are stored for reuse across exports

### Storage Quota

- View used vs. available storage
- See total asset count
- Delete unused assets to free space

---

## Best Practices

### For Demo Videos

1. Use **Desktop** presentation mode with a clean background
2. Add intro/outro cards with your branding
3. Set **frame duration** to 2-3 seconds for readability
4. Choose **Ease In/Out** cursor speed for smooth motion

### For Bug Reports

1. Use **Browser Frame** mode to show the full context
2. Disable intro/outro cards for brevity
3. Export as **MP4** for easy sharing
4. Keep **Auto** render source for quality

### For Social Media

1. Export as **GIF** for easy embedding
2. Use **720p** dimensions for faster loading
3. Enable **Loop** for continuous playback
4. Keep it short (5-10 frames maximum)

### For Documentation

1. Use **Fullscreen** mode to focus on content
2. Add **watermark** for brand recognition
3. Export as **MP4** for broad compatibility
4. Consider **1080p** for print-quality screenshots
`;
