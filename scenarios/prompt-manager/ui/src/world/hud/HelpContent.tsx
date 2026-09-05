import type { CameraTuning } from '../config'

/** In-app help reads the active input map, including live lever overrides. */
export function WorldHelpContent({ camera }: { camera: CameraTuning }) {
  const label = (action: string) => action.replace(/-/g, ' + ')
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
        Mouse: left drag {label(camera.input.mouse.left)}, middle drag {label(camera.input.mouse.middle)},
        right drag {label(camera.input.mouse.right)}, wheel {label(camera.input.mouse.wheel)}.
        Touch: one finger {label(camera.input.touch.one)}, two fingers {label(camera.input.touch.two)},
        three fingers {label(camera.input.touch.three)}. Truck means pan.
        Zoom {camera.dollyToCursor ? 'follows the pointer' : 'uses the orbit target'}.
        Arrow keys orbit, +/- zoom, <kbd className="rounded border border-border px-1 text-xs">Esc</kbd>{' '}
        returns home. Toggle 2D in the Swarm panel to use the same actions without the canvas.
      </p>
    </div>
  )
}
