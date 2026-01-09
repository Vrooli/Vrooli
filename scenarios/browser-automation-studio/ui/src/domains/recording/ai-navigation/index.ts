/**
 * AI Navigation Module
 *
 * Core hooks and components for AI-driven browser navigation.
 * The main UI is provided by the unified sidebar's AutoTab component.
 */

export {
  AINavigationStepSkeleton,
  AINavigationTimelineSkeleton,
  AIProcessingIndicator,
  WebSocketConnectingIndicator,
} from './AINavigationSkeleton';
export { HumanInterventionOverlay } from './HumanInterventionOverlay';
export { useAINavigation } from './useAINavigation';
export * from './types';
