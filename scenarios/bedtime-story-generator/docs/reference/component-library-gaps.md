# Bedtime Story Generator component-library gaps

Bedtime Story Generator is a full-viewport immersive story room. Three.js
camera rails, scene controls, time-of-day state, reader and generator panels,
and parent/developer overlays are contextual controls over the scene rather
than application-destination navigation. `AmbientDisplayShell` is the closest
library archetype, but it is pre-1.0 and cannot carry this interaction model
without changing the experience. The working scene remains scenario-owned
until a proven immersive archetype exists.

```shell-ejection
{"archetype":"ambient-display","reason":"Bedtime Story Generator is a full-viewport Three.js story room whose camera rails, scene controls, time-of-day state, reader/generator panels, and parent/developer overlays are contextual to the immersive scene. AppShell/2 destination navigation would replace that interaction model, while AmbientDisplayShell is pre-1.0 and cannot carry it; retain the working scene under a scoped ejection.","files":["ui/src/App.jsx","ui/src/components/ChildrensRoom.jsx","ui/src/components/SceneControls.jsx","ui/src/components/BookReader.jsx","ui/src/components/StoryGenerator.jsx","ui/src/components/SettingsPanel.jsx","ui/src/components/SceneDebugPanel.jsx","ui/src/ParentDashboard.jsx"]}
```
