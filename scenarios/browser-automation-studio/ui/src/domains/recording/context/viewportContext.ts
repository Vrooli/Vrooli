import { createContext } from 'react';
import type { ViewportContextValue } from './ViewportProvider';

export const ViewportContext = createContext<ViewportContextValue | null>(null);
