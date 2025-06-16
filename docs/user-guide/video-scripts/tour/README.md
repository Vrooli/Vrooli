# Tour Video Script & Generation Guide

Style: Empowering, inspiring, and informative.
Audience: Potential users, general audience.
Colors: Green,  blue, black, neon
Buzzword: "AI agents"
Length: ~60 seconds.

## Links

- [Synthesia website (for video generation)][Synthesia]
- [Rotato website (for fancy footage of app displayed on phone/laptop)][Rotato]
- [Cursorful extension (for screen recording)][Cursorful]
- [Veo 3 for video generation (preferred)][Veo-3]
- [Sora for video generation (alternative - good for making a video loop)][Sora]

---

## Slide-by-Slide Script & Scene Map

Use the scene map to design the video in [Synthesia][Synthesia]. Paste the script into each scene, and let Synthesia generate the video. It's that simple!  

**Note**: Slides 1 and 3 can be swapped depending on the audience or through A/B testing. The rest of the slides are generic enough to work for any audience.

Some alternative ideas for slides 1 and 3:
- Other solo-founder product launches
- Academic research
- Employee onboarding
- Non-profit fundraising campaign
- Financial forecasting
- Revenue Operations (RevOps) growth sprint

### 1. Hook

**Template 1**: Video-background slide (~2 seconds)  
**Media 1**: End of `clip02_swarm_chat_view.mp4` (swarm finishing)  

**Template 2**: Video-background slide  
**Media 2**: `clip01_gradient_loop.mp4` with logo overlay

> "Have an idea but don't want to do the work? Meet Vrooli."

### 2. Personal Welcome

**Template**: Narrator close-up  
**Media**: *None*  

> "Imagine a team of specialists in your pocket - always ready for any challenge."

### 3. Demonstrate Value

**Template 1**: Video on left (2/3), narrator on right (1/3)
**Media 1**: `clip02_swarm_chat_view.mp4`  

**Template 2**: Video-background slide (starting at "By lunch")
**Media 2**: `clip02_swarm_chat_view.mp4`

**Note**: We use 2 templates here solely to make the video more engaging.

> "Over breakfast you decide to launch a coffee subscription business.  
>  
>  By lunch, our AI agents line up suppliers, licenses, branding, and logistics — no experience needed."

### 4. Introduce Self-Improvement

**Template**: Video-background slide  
**Media**: `clip03_self_improve.mp4`

> "Our agents learn from experience and rewrite themselves, so that you wield more power each day."

### 5. The Grand Vision

**Template**: Video-background slide  
**Media**: `clip04_grand_vision.mp4`

> "We're building an open-source economy where businesses are automated from the top down.  

### 6. Call-to-Action

**Template**: Narrator close-up
**Media**: *None*

> "Launch your first AI swarm — free, in under a minute."  
>  
> "Big ideas deserve big teams — yours starts now."

### 7. Logo

**Template**: Video-background slide  
**Media**: `clip01_gradient_loop.mp4` with logo overlay

*No audio. Solid for 3 seconds, then quick fade to black.*

---

## B-Roll & Screen-Recording Checklist

### 1. `clip01_gradient_loop.mp4`

**Prompt**: Dark futuristic backdrop with slow diagonal magenta-cyan gradient light sweep, tiny dust particles drifting. Soft synth-pad drone underscoring the motion.

### 2. `clip02_swarm_chat_view.mp4`

**Steps**:  
1. Navigate to `/chat`.  
2. Start recording using the [Cursorful][Cursorful] extension.
3. Click input box, type: "Create a zero-waste coffee subscription business" and press Enter.  
4. Record for a few minutes.
5. Stop recording.
6. Trim and speed up the video until you have a ~15 second clip. Add fast-forward icon to corner so as not to mislead the viewer.
7. Use [Rotato][Rotato] to display the video on a phone/laptop in a stunning way.

### 3. `clip03_self_improve.mp4`

**Prompt**: Create a high-definition top-down shot of an online environment populated with glowing hexagonal nodes resembling AI agents. These nodes emit a soft neon glow—some pulse bright red rhythmically, emphasizing their activity. The scene is set in a sleek, futuristic environment with dark, subtle backgrounds contrasting the bright lights. Incorporate gentle camera tilt-down movement to emphasize the nodes’ spatial arrangement. Begin with a wide-angle view: the nodes are scattered in a seemingly chaotic pattern. Suddenly, select nodes pulse brighter red, signaling a brief alert; their glow intensifies with a soft, rhythmic "whoosh" sound effect, enhancing the tension. Following this, the nodes collectively halt their red pulses and pause momentarily, with the camera subtly zooming in to focus on their group behavior. Then, coordinate a smooth, synchronized transformation: the nodes rapidly realign into an organized, efficient hexagonal lattice structure. As they rearrange, their glow shifts from red to a calming, luminous green, pulsing gently, signifying optimized communication and harmony. Overlay a soft, ambient electronic soundtrack with subtle whooshing sounds that synchronize with their shifting positions. Use cinematic lighting emphasizing neon effects—glowing edges, soft shadows, and subtle reflections on the virtual surface—to enhance the futuristic aesthetic. Employ slow, deliberate camera movements—like a steady pan or orbit—to follow the process, culminating in a stable shot of the ordered lattice glowing green. Include visual cues of energy flow between nodes—thin, luminous lines connecting adjacent agents—to deepen the sense of interconnected system dynamics. Overall, create a vivid, energetic visualization of AI nodes reacting and reconfiguring, combining precise animation, synchronized sound effects, and a sleek, technological style to evoke a sense of emergent intelligence and organized harmony.

### 4. `clip04_grand_vision.mp4`

**Prompt**: Cyber-green megacity at night. Rain glistens on neon-lit street. Pedestrians in reflective smart fabrics pass humanoid service robots holding umbrellas; silent e-cars glide through the puddles.

Camera lifts in a slow crane-up and gentle pull-back. It rises above the crowd to reveal treetop sky-bridges and a few cyan-ring e-VTOL taxis lifting off nearby roofs.

Wide aerial finale. Animated circuit-lines pulse across vertical-garden façades; drones trace soft emerald arcs above the skyline as the city glows in serene equilibrium.

Audio: light rain, soft synthwave bed, distant e-motor hum that fades to faint drone whir as the camera ascends.

---

## Screen Recording Tips

- Record normal videos without any fancy effects using the Snipping Tool.
- Display the normal scrren recordings on a phone/laptop using [Rotato][Rotato]. Pick a few different templates to show off the app on different devices
- For recordings you don't want displayed on a device (will have fancy effects), use [Cursorful][Cursorful] on a fresh window (no other tabs open unless that's part of the scene) with the taskbar hidden.

---

## Synthesia Production Tips

- Paste each script block into its own Synthesia scene; drop B-roll with `{VIDEO: clipname.mp4}` tokens.
- Circle avatar overlay for UI shots; full-frame avatar for slides 2 & 12.
- Enable burned-in captions—many viewers watch muted.
- Use brand font, white text, accent color #0FA for highlights.
- Place a “Start for free” button directly below the video on the landing page.

[Synthesia]: https://app.synthesia.io/
[Rotato]: https://rotato.app/
[Cursorful]: https://cursorful.com/editor
[Veo-3]: https://gemini.google.com/app
[Sora]: https://sora.chatgpt.com/