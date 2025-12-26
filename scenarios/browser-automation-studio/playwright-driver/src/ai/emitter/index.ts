/**
 * Emitter Module
 *
 * This module provides event emission for navigation steps to the API.
 */

// Types
export * from './types';

// Callback Emitter
export {
  createCallbackEmitter,
  emitNavigationComplete,
  createMockEmitter,
  type CallbackEmitterConfig,
} from './callback-emitter';
