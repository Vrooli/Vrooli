/**
 * Frame Streaming Strategy Interface
 *
 * Defines the contract for different frame streaming implementations.
 * Currently supports CDP screencast (preferred) and polling (fallback).
 *
 * @module frame-streaming/strategies/interface
 */

import type { Page } from 'playwright';
import type { FrameWebSocket } from '../types';

/**
 * Configuration for starting a streaming strategy.
 */
export interface StreamingStrategyConfig {
  /** Session ID for logging and metrics */
  sessionId: string;
  /** Frame quality 1-100 */
  quality: number;
  /** Target FPS (primarily used by polling strategy) */
  targetFps: number;
  /** Screenshot scale */
  scale: 'css' | 'device';
  /** Whether to include performance timing headers */
  includePerfHeaders: boolean;
}

/**
 * Callback to get WebSocket connection state.
 * Strategy implementations use this to check if WS is ready before sending frames.
 */
export interface WebSocketProvider {
  /** Get current WebSocket (may be null if not connected) */
  getWebSocket(): FrameWebSocket | null;
  /** Check if WebSocket is ready to send */
  isReady(): boolean;
}

/**
 * Callback to report frame statistics.
 */
export interface FrameStatsReporter {
  /** Report a successfully sent frame */
  onFrameSent(stats: FrameStats): void;
  /** Report a skipped frame */
  onFrameSkipped(reason: 'unchanged' | 'timeout' | 'ws_not_ready'): void;
}

/**
 * Statistics for a single frame.
 */
export interface FrameStats {
  /** Capture time in ms */
  captureMs: number;
  /** Frame comparison time in ms (polling only) */
  compareMs?: number;
  /** WebSocket send time in ms */
  wsSendMs: number;
  /** Frame size in bytes */
  frameBytes: number;
}

/**
 * Handle returned when starting a streaming strategy.
 */
export interface StreamingHandle {
  /** Stop the streaming and cleanup resources */
  stop(): Promise<void>;
  /** Get total frames sent */
  getFrameCount(): number;
  /** Check if streaming is currently active */
  isActive(): boolean;
  /** Update quality setting (if supported) */
  updateQuality?(quality: number): void;
  /** Update target FPS (polling only) */
  updateTargetFps?(fps: number): void;
}

/**
 * Interface for frame streaming strategies.
 *
 * Strategies handle the mechanics of capturing and sending frames.
 * The orchestrator handles WebSocket lifecycle and strategy selection.
 */
export interface FrameStreamingStrategy {
  /** Human-readable name for logging */
  readonly name: string;

  /**
   * Start streaming frames from the page.
   *
   * @param page - Playwright page to capture
   * @param config - Streaming configuration
   * @param wsProvider - Provider for WebSocket connection
   * @param statsReporter - Reporter for frame statistics
   * @returns Handle to control and stop streaming
   */
  start(
    page: Page,
    config: StreamingStrategyConfig,
    wsProvider: WebSocketProvider,
    statsReporter: FrameStatsReporter
  ): Promise<StreamingHandle>;

  /**
   * Check if this strategy is supported for the given page/browser.
   * CDP screencast requires Chrome/Chromium, polling works everywhere.
   */
  isSupported(page: Page): Promise<boolean>;
}
