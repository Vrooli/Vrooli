# Tour Video Script & Generation Guide

---

## Slide-by-Slide Script & Scene Map

### 1. Hook

*Template block*: Hook (logo + motion background)
*Script*:

> “Ever wished you could hand your to-do list to a squad of tireless AIs and watch it melt away? Stick around—you’re about to see how.”

### 2. Personal Welcome

*Template block*: Narrator close-up
*Script*:

> “Hi, I’m Lia - your guide. Think of Vrooli like a team of specialists in your pocket, always on shift, always up for a challenge.”

### 3. The Problem

*Template block*: Video-background slide
*Script*:

> “Most apps make you bounce between tabs, copy data, and pray nothing breaks. That grind slows projects—and people—down.”

### 4. The Solution

*Template block*: Clean mock-up slide
*Script*:

> “With Vrooli, you just chat. Say, ‘Draft a marketing plan for my eco-shoe launch,’ hit send, and our agents spin up the research, copy, and visuals for you.”

### 5. Conversational Proof

*Template block*: Two-avatar dialogue
*Script*:

> **Avatar A:** “Need Spanish copy, too?”
> **Avatar B:** “Already translating—language agent’s on it.”

### 6. Key Benefit #1

*Template block*: Gradient benefit slide
*Script*:

> “It’s *fast.* Swarm coordination means tasks finish in minutes—not days.”

### 7. Key Benefit #2

*Template block*: Gradient benefit slide
*Script*:

> “It’s *getting smarter.* Every routine learns from the last run, so your results keep leveling up while you sleep.”

### 8. Key Benefit #3

*Template block*: Gradient benefit slide
*Script*:

> “And it’s *safe.* End-to-end encryption plus sandboxed execution keep your data locked tight.”

### 9. Feature Focus

*Template block*: Video-insert slide
*Script*:

> “Here’s the Routine Builder. Drag steps, nest sub-routines, link an API—then watch the agents test it live.”

### 10. Four-Point Summary

*Template block*: Icon grid
*Script*:

> • Swarms in one click
> • Universal workflow engine
> • Built-in credit control
> • Mobile & API ready

### 11. Call-to-Action

*Template block*: Mobile mock-up
*Script*:

> “Jump in for free today. Type your first request and meet your new AI coworkers.”

### 12. Warm Close

*Template block*: Video-background slide
*Script*:

> “Thanks for watching. Ready to run at billionaire speed? Let’s get started.”

---

## B-Roll & Screen-Recording Checklist

*(Clip names in brackets; insert in Synthesia with `{VIDEO: clip_name.mp4}`)*

### Universal Specs

* 1080p, 30 fps, 16 × 9, MP4
* Keep cursor movements slow and deliberate; zoom to 125 % if text is small.
* Record each clip a few seconds longer than needed—trim in post.

---

### 1. Split-Screen Productivity (Slides 1 & 3) – 5 s `[clip01_before_after.mp4]`

*Left half*: Open four desktop apps, frantically alt-tab, copy-paste text; sigh.
*Right half*: Vrooli chat UI with a single request being processed (type and hit Enter, first agent reply appears).
*How*: Capture two separate recordings, compose into split-screen in your editor, or use OBS “Side-by-Side” layout.

### 2. Frustration-Reaction Stock Footage (Slide 3) – 6 s `[clip02_frustration_stock.mp4]`

*Royalty-free or AI-generated prompt*: “Office worker rubbing temples while staring at cluttered computer screen, cinematic lighting.”
*Source*: Pexels / Pixabay, or generate with Runway Gen-2.

### 3. Live Chat Demo (Slide 4) – 10 s `[clip03_chat_marketing_plan.mp4]`

*Steps*:

1. Navigate to `/chat`.
2. Click input box, type: *Draft a marketing plan for an eco-shoe launch* and press Enter.
3. Record until first agent message and spinner appear (≈ 10 s).

### 4. Conversational Translation (Slide 5) – 7 s `[clip04_translation_queue.mp4]`

*Steps*:

1. Same chat thread, scroll to show language agent automatically queuing Spanish copy.
2. Highlight “Queued by Language Agent” badge.

### 5. Routine Builder Walk-Through (Slide 9) – 12 s `[clip05_routine_builder.mp4]`

*Steps*:

1. Open `/routine/new`.
2. Drag three nodes: “Research”, “Write Copy”, “Generate Images”.
3. Nest a sub-routine under “Write Copy”.
4. Click **Run**, show live log stream scrolling.

### 6. Dashboard Overview (Slide 6) – 6 s `[clip06_dashboard_hover.mp4]`

*Steps*:

1. Open `/dashboard`.
2. Hover over **Active Projects** card (tooltip pops).
3. Move to **Smart Suggestions** card (tooltip pops).

### 7. Security Settings Toggle (Slide 8) – 6 s `[clip07_security_toggle.mp4]`

*Steps*:

1. Navigate to `/settings/security`.
2. Toggle **Encryption** switch ON (green check).
3. Toast “HIPAA Ready” badge slides in.

### 8. Swarm Creation Flow (Slide 6 or overlay later) – 8 s `[clip08_assemble_swarm.mp4]`

*Steps*:

1. Open `/swarms/new`.
2. Search “marketing”, select three agents.
3. Click **Assemble Swarm**, show confirmation toast.

### 9. Mobile PWA Scroll (Slide 11) – 6 s `[clip09_mobile_scroll.mp4]`

*Steps*:

1. In Chrome DevTools, emulate iPhone 14; load `https://vrooli.com`.
2. Swipe through tabs **Chat → Swarm → Routines**.

### 10. Ambient Gradient Loop (Slides 1 & 12) – 10 s `[clip10_gradient_loop.mp4]`

*AI prompt (if generating)*: “Seamless looping neon gradient animation, dark background with magenta and cyan diagonal motion, subtle, futuristic.”
*Ensure*: Loopable, no text; export as 1080p MP4.

---

## Synthesia Production Tips

* Limit each scene to ≤ 20 s; target total runtime ≈ 3–4 minutes.
* Paste each script block into its own Synthesia scene; drop B-roll with `{VIDEO: clipname.mp4}` tokens.
* Circle avatar overlay for UI shots; full-frame avatar for slides 2 & 12.
* Enable burned-in captions—many viewers watch muted.
* Use brand font, white text, accent color #0FA for highlights.
* Place a “Start for free” button directly below the video on the landing page.

---

## Next Steps

1. Copy each script segment into Synthesia, one scene per slide.
2. Record the nine B-roll clips (OBS or CleanShot, 1080p 30 fps).
3. Import clips, choose avatars, apply brand styling, add captions.
4. Publish, embed on landing page, and monitor completion & CTA-click metrics; iterate if drop-off > 30 % before slide 8.
