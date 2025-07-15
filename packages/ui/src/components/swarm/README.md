# Swarm Components

This directory contains components for displaying and controlling AI swarms in the chat interface.

## Components

### SwarmPlayer
A compact "media player" style component that shows swarm status and controls. It displays:
- Current swarm status (Running, Paused, Failed, etc.)
- Goal/objective text (truncated)
- Credits used
- Control buttons (pause/play/stop)
- Progress bar based on subtask completion

Clicking the player opens the detailed SwarmDetailPanel in the right drawer.

### SwarmDetailPanel
A comprehensive panel that displays detailed swarm information including:
- **Overview**: Statistics, elapsed time, credits used, pending approvals
- **Tasks Tab**: List of subtasks with status, assignees, and dependencies
- **Agents Tab**: Swarm leader, team members, and subtask assignments
- **Resources Tab**: Files, notes, and other resources created by the swarm
- **Approvals Tab**: Pending tool calls requiring user approval
- **History Tab**: Execution history of routines and tool calls

## Integration

The SwarmPlayer is integrated into the ChatCrud view, appearing between the chat messages and input box. When a swarm is active for the current chat, the player will automatically display.

The SwarmDetailPanel is shown in the right drawer when the player is clicked. The drawer content is managed by the RightDrawerContent component which switches between ChatCrud and SwarmDetailPanel based on PubSub events.

## TODO

1. **Server Integration**: 
   - Fetch swarm configuration from server based on chat ID
   - Implement API calls for pause/resume/stop operations
   - Handle tool approval/rejection via API

2. **Real-time Updates**:
   - Subscribe to swarm state changes via WebSocket
   - Update progress and status in real-time

3. **Translations**:
   - Add all translation keys to the translation files
   - Support multi-language display

4. **Persistence**:
   - Store swarm configuration in chat metadata
   - Persist swarm state across page refreshes

## Usage Example

```tsx
import { SwarmPlayer } from "./components/swarm";

// In your chat component
<SwarmPlayer
  swarmConfig={chatSwarmConfig}
  swarmStatus={currentSwarmStatus}
  onPause={handlePauseSwarm}
  onResume={handleResumeSwarm}
  onStop={handleStopSwarm}
/>
```