/**
 * Frame Streaming Strategies
 *
 * This module exports all streaming strategy implementations.
 *
 * @module frame-streaming/strategies
 */

// Strategy interface
export type {
  FrameStreamingStrategy,
  StreamingStrategyConfig,
  StreamingHandle,
  WebSocketProvider,
  FrameStatsReporter,
  FrameStats,
} from './interface';

// Strategy implementations
export { CdpScreencastStrategy, createCdpScreencastStrategy } from './cdp-screencast';
export { PollingStrategy, createPollingStrategy } from './polling';
