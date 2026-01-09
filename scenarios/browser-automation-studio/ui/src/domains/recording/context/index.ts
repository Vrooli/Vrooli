/**
 * Recording Context exports
 *
 * Provides React context providers for centralized state management
 * in the recording domain.
 */

export {
  ViewportProvider,
  useViewport,
  useViewportOptional,
  type ViewportProviderProps,
  type ViewportContextState,
  type ViewportContextActions,
  type ViewportContextValue,
  type ViewportDimensions,
  type ViewportSyncState,
} from './ViewportProvider';
