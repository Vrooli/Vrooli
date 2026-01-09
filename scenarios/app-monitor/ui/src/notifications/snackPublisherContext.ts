import { createContext } from 'react';
import type { SnackPublishOptions, SnackUpdateOptions } from './snackBus';

export interface SnackPublisherApi {
  publish(options: SnackPublishOptions): string;
  replace(id: string, options: SnackPublishOptions): void;
  patch(id: string, changes: SnackUpdateOptions): void;
  dismiss(id: string): void;
  clear(): void;
}

export const SnackPublisherContext = createContext<SnackPublisherApi | null>(null);
