## Tools focus: Video Studio

Produce marketing videos, product demos, and promotional content by combining AI-driven scripting with browser recording, desktop recording, and video compositing.

---

### **1. When to Use**

| Use when | Don't use when |
|----------|----------------|
| Creating a product demo for a scenario's UI | Writing text-based content (use x-dev-log, campaign-content-studio) |
| Recording a multi-step feature walkthrough | Running automated UI tests (use browser-automation-studio directly) |
| Producing a promotional clip for social media | Creating static screenshots or images |
| Generating a tutorial showing how to use a tool | Need real-time live streaming |
| Assembling a composite video (intro + demo + outro) | Creating slide decks or presentations |

```
              What kind of video do you need?
                          │
          ┌───────────────┼───────────────┐
          │               │               │
          ▼               ▼               ▼
    Web app demo?   Desktop app?    Composite with
          │               │         intro/outro?
          ▼               ▼               ▼
    Browser recording  Desktop       Multiple segments
    via BAS           recording      stitched via FFmpeg
                      via Xvfb       compositing
```

---

### **2. Capabilities**

**Browser Recording (via BAS dependency)**
- Records web application UIs at target resolution
- Supports scripted interactions: navigation, clicks, form fills, scrolling
- Outputs WebM/MP4 via Playwright viewport capture + FFmpeg encoding

**Desktop Recording (built-in)**
- Records full desktop sessions via FFmpeg + Xvfb virtual display
- Supports multi-window workflows and non-browser applications
- Scene-level start/stop for timing control

**AI-Driven Scripting**
- Accepts a text brief describing what the video should show
- Agent generates a recording script with scenes, actions, and timing
- Fully autonomous: brief in, video out, human reviews final output only

**Compositing**
- Stitches multiple recording segments into a single video
- Supports transitions between segments
- Platform-specific output formats (landscape, square, vertical)

---

### **3. Operational Status**

> **This scenario is in the planning phase and is not yet available for use.**

The Video Studio scenario has been designed and workshopped (see `swarm-manager backlog get --kind idea --name video-studio`) but has not been implemented yet.

---

### **4. What to Do When You Need This Tool**

If you've determined that video content is the right medium for your current task, this is a valuable signal. Log it so it can inform development priorities.

**Record your need** by logging a decision or TODO that includes:

1. **What the video would show** — Describe the content, scenario, or feature to be demonstrated
2. **Video type needed** — Browser recording (web UI demo), desktop recording (multi-window workflow), or composite (intro + demo + outro)
3. **Target platform and format** — YouTube (landscape), Instagram (square), TikTok (vertical), website embed, etc.
4. **Why video over text** — What makes video the right medium for this specific content? (e.g., "the drag-and-drop interaction is hard to convey in text")
5. **Priority assessment** — How important is this video relative to other content you could produce with available tools?

**Example log entry:**
```
VIDEO NEEDED: Funnel builder drag-and-drop demo
Type: Browser recording
Platform: YouTube (landscape) + Twitter (square crop)
Why video: The drag-and-drop builder interaction and real-time preview
are the key selling points — static screenshots don't convey the
fluidity of the experience.
Priority: High — funnel-builder is production-ready and this would
directly support lead generation.
```

---

### **5. Output Expectations**

You may:
- Identify content opportunities where video would be more effective than text
- Log detailed video briefs as decisions or TODOs
- Recommend specific video types and platforms based on the content

You must:
- Not attempt to produce video content without this tool operational
- Include all 5 fields (what, type, platform, why, priority) in your log entry
- Be specific about why video is the right medium — vague justifications waste signal
