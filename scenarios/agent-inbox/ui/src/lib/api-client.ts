/**
 * API Client Interface for Dependency Injection
 *
 * This module defines interfaces for the API client, enabling:
 * - Dependency injection in hooks for easier testing
 * - Mocking of API calls in unit tests
 * - Type-safe API client implementations
 *
 * @example
 * // In tests, create a mock client:
 * const mockClient: CompletionClient = {
 *   completeChat: jest.fn().mockImplementation(async (chatId, options) => {
 *     options?.onEvent?.({ type: 'content', content: 'Hello' });
 *   }),
 *   approveToolCall: jest.fn().mockResolvedValue({ success: true }),
 *   rejectToolCall: jest.fn().mockResolvedValue(undefined),
 * };
 *
 * // In production, use the default client:
 * import { defaultCompletionClient } from './api-client';
 *
 * @see useCompletion - Hook that uses this interface
 * @see docs/SEAMS.md - Testing seams documentation
 */

import type { StreamingEvent, ApprovalResult, SkillPayloadForAPI } from "./api";

// =============================================================================
// Completion Client Interface
// =============================================================================

/**
 * Options for completion requests.
 */
export interface CompletionOptions {
  /** Whether to stream the response (default: true) */
  stream?: boolean;
  /** Callback for streaming events */
  onEvent?: (event: StreamingEvent) => void;
  /** AbortSignal for cancellation */
  signal?: AbortSignal;
  /** Skills to inject into tool context */
  skills?: SkillPayloadForAPI[];
}

/**
 * Interface for completion-related API calls.
 *
 * This interface abstracts the completion API for testing:
 * - completeChat: Main streaming completion
 * - approveToolCall: Approve a pending tool call
 * - rejectToolCall: Reject a pending tool call
 *
 * @example
 * // Create a mock for testing
 * const mockClient: CompletionClient = {
 *   completeChat: jest.fn().mockImplementation(async (chatId, options) => {
 *     // Simulate streaming content
 *     options?.onEvent?.({ type: 'content', content: 'Test' });
 *     // Simulate completion
 *     options?.onEvent?.({ type: 'progress', done: true });
 *   }),
 *   approveToolCall: jest.fn().mockResolvedValue({
 *     success: true,
 *     tool_result: { id: '1', tool_name: 'test', status: 'completed' },
 *     pending_approvals: [],
 *     auto_continued: true,
 *   }),
 *   rejectToolCall: jest.fn().mockResolvedValue(undefined),
 * };
 */
export interface CompletionClient {
  /**
   * Run a chat completion with optional streaming.
   *
   * @param chatId - Chat to complete
   * @param options - Completion options
   * @returns Promise that resolves when streaming completes
   */
  completeChat(chatId: string, options?: CompletionOptions): Promise<void>;

  /**
   * Approve a pending tool call for execution.
   *
   * @param toolCallId - ID of the tool call to approve
   * @param chatId - Chat ID for validation
   * @returns Approval result with execution details
   */
  approveToolCall(toolCallId: string, chatId: string): Promise<ApprovalResult>;

  /**
   * Reject a pending tool call.
   *
   * @param toolCallId - ID of the tool call to reject
   * @param chatId - Chat ID for validation
   * @param reason - Optional rejection reason
   */
  rejectToolCall(toolCallId: string, chatId: string, reason?: string): Promise<void>;
}

// =============================================================================
// Default Implementation
// =============================================================================

import { completeChat, approveToolCall, rejectToolCall } from "./api";

/**
 * Default completion client using the real API functions.
 *
 * Use this in production code. For testing, create a mock implementation
 * of the CompletionClient interface instead.
 */
export const defaultCompletionClient: CompletionClient = {
  completeChat: async (chatId, options) => {
    await completeChat(chatId, options);
  },
  approveToolCall,
  rejectToolCall,
};

// =============================================================================
// Context for Dependency Injection (Optional Pattern)
// =============================================================================

import { createContext, useContext } from "react";

/**
 * React context for providing a completion client.
 *
 * This enables dependency injection at the React tree level:
 *
 * @example
 * // In your app root (production):
 * <CompletionClientProvider value={defaultCompletionClient}>
 *   <App />
 * </CompletionClientProvider>
 *
 * @example
 * // In tests:
 * <CompletionClientProvider value={mockClient}>
 *   <ComponentUnderTest />
 * </CompletionClientProvider>
 */
export const CompletionClientContext = createContext<CompletionClient>(defaultCompletionClient);

/**
 * Hook to access the completion client from context.
 *
 * @returns The completion client from context (defaults to real API)
 *
 * @example
 * function MyComponent() {
 *   const client = useCompletionClient();
 *   // Use client.completeChat(), client.approveToolCall(), etc.
 * }
 */
export function useCompletionClient(): CompletionClient {
  return useContext(CompletionClientContext);
}

/**
 * Provider component for the completion client.
 *
 * @example
 * // Wrap your app or test component
 * <CompletionClientProvider value={myClient}>
 *   <ChildComponents />
 * </CompletionClientProvider>
 */
export const CompletionClientProvider = CompletionClientContext.Provider;
