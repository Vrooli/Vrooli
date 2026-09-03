/** In-app help for the world; mirrors docs/concepts/WORLD-HUD.md. */
export function WorldHelpContent() {
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
        Drag to orbit, scroll to dolly, arrow keys orbit, <kbd className="rounded border border-border px-1 text-xs">Esc</kbd>{' '}
        returns home. Toggle 2D in the Swarm panel to use the same actions without the canvas.
      </p>
    </div>
  )
}
