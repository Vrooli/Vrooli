import type { ReactNode } from 'react';
import { MetricsModeContext, type MetricsMode } from './MetricsModeContext';

export function MetricsModeProvider({
  mode = 'live',
  children,
}: {
  mode?: MetricsMode;
  children: ReactNode;
}) {
  return <MetricsModeContext.Provider value={mode}>{children}</MetricsModeContext.Provider>;
}
