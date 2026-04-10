/** Module-level cursor ref — written by React event handlers, read by useFrame in agents. Zero re-renders. */
export const cursorRef = { current: null as { x: number; y: number } | null }
