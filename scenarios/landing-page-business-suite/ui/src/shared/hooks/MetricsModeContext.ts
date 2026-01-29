import { createContext } from 'react';

export type MetricsMode = 'live' | 'preview';

export const MetricsModeContext = createContext<MetricsMode>('live');
