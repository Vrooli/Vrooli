import type { CameraTuning } from '../config'

/** In-app help reads the active input map, including live lever overrides. */
export function WorldHelpContent({ camera }: { camera: CameraTuning }) {
  const label = (action: string) => action.replace(/truck/g, 'pan').replace(/dolly/g, 'zoom').replace(/rotate/g, 'orbit').replace(/-/g, ' + ')
  return (
    <div className="space-y-3 text-sm text-muted-foreground">
      <p>
        Each blob is one of your agents. Where it stands is its state: at its desk when a run is active (spinning ring),
        at the team table when a heartbeat is due (amber marker), in the commons when idle. A red marker means the last
        run failed.
      </p>
      <p>
        The strip at the top counts running, gathering, idle and failed agents and shows the next heartbeat; click a count
        to filter. The Swarm panel lists teams and recent events. Click an agent to open its card: Run now, Stop,
        Acknowledge, Open editor, Follow.
      </p>
      <p>
        Use the Camera toolbar to choose Orbit drag or Pan drag, frame a selection, or switch to top, front, and isometric views.
        Mouse: left drag uses the selected tool, middle drag {label(camera.input.mouse.middle)},
        right drag {label(camera.input.mouse.right)}, wheel zooms. Choose Trackpad in Input settings for two-finger scroll to pan and pinch to zoom.
        Touch: one finger {label(camera.input.touch.one)}, two fingers {label(camera.input.touch.two)},
        three fingers {label(camera.input.touch.three)}.
        Zoom {camera.dollyToCursor ? 'follows the pointer' : 'uses the orbit target'}.
        Click the world to use the keyboard. Arrow keys orbit, WASD pans, +/- zooms, <kbd className="rounded border border-border px-1 text-xs">Esc</kbd>{' '}
        returns home. Toggle 2D in the Swarm panel to use the same actions without the canvas.
      </p>
      <p>First person and Third person let you walk as a separate visitor: WASD or arrows move, Shift runs, Space jumps, and dragging looks around.
        Switching modes focuses the world. Low obstacles can be stepped over. Click an agent to invite them over to face you; End conversation releases them.
        Capture mouse enables continuous look; Escape releases it. Return to Explore restores your previous inspection view.
        Walls, furniture, water, steep ground, and the world edge limit walking. Follow tracks an agent without taking control of it.</p>
    </div>
  )
}
