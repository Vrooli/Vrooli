import * as React from 'react';
import AppRouter from './AppRouter';
import { useExecutionUpdates } from './hooks/useExecutionUpdates';
import { ensureReadyMarker, markAppReady } from './ready';

export function ReadyMarker(): null {
  React.useEffect(() => {
    ensureReadyMarker();
    // Slightly defer setting the ready flag until after first paint.
    requestAnimationFrame(() => markAppReady());
  }, []);
  return null;
}

// Wrapper component that enables WebSocket-based real-time updates
export function AppWithUpdates(): React.ReactElement {
  // Listen to WebSocket messages and update stores accordingly
  useExecutionUpdates();
  return <AppRouter />;
}
