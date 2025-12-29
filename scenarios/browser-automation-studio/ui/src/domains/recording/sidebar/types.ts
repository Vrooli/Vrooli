/**
 * Sidebar Domain Types
 *
 * Types for the unified sidebar with Timeline and Auto tabs.
 */

import type { AINavigationStep, HumanInterventionState, TokenUsage } from '../ai-navigation/types';

// ============================================================================
// Tab State Types
// ============================================================================

/**
 * Available sidebar tabs.
 */
export type TabId = 'timeline' | 'auto';

/**
 * State for the unified sidebar.
 */
export interface UnifiedSidebarState {
  activeTab: TabId;
  width: number;
  isResizing: boolean;
  timelineActivity: boolean;
  autoActivity: boolean;
}

// ============================================================================
// AI Settings Types
// ============================================================================

/**
 * AI navigation settings stored in localStorage.
 */
export interface AISettings {
  model: string;
  maxSteps: number;
}

/**
 * Default AI settings.
 */
export const DEFAULT_AI_SETTINGS: AISettings = {
  model: 'qwen3-vl-30b',
  maxSteps: 20,
};

// ============================================================================
// AI Message Types (Chat Interface)
// ============================================================================

/**
 * Message role in the AI conversation.
 */
export type AIMessageRole = 'user' | 'assistant' | 'system';

/**
 * Status of an AI message/navigation.
 */
export type AIMessageStatus = 'pending' | 'running' | 'completed' | 'failed' | 'aborted' | 'awaiting_human';

/**
 * A message in the AI conversation.
 */
export interface AIMessage {
  /** Unique message ID */
  id: string;
  /** Who sent the message */
  role: AIMessageRole;
  /** Message content (prompt for user, summary for assistant) */
  content: string;
  /** When the message was created */
  timestamp: Date;
  /** Navigation ID if this message triggered navigation */
  navigationId?: string;
  /** Current status of navigation (for assistant messages) */
  status?: AIMessageStatus;
  /** Steps completed during this navigation */
  steps?: AINavigationStep[];
  /** Total tokens used */
  totalTokens?: number;
  /** Error message if navigation failed */
  error?: string;
  /** Whether abort button should be shown */
  canAbort?: boolean;
  /** Human intervention state if awaiting human */
  humanIntervention?: HumanInterventionState;
}

/**
 * Create a user message.
 */
export function createUserMessage(content: string): AIMessage {
  return {
    id: `user-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
    role: 'user',
    content,
    timestamp: new Date(),
  };
}

/**
 * Create a pending assistant message.
 */
export function createAssistantMessage(navigationId: string): AIMessage {
  return {
    id: `assistant-${navigationId}`,
    role: 'assistant',
    content: '',
    timestamp: new Date(),
    navigationId,
    status: 'pending',
    steps: [],
    totalTokens: 0,
    canAbort: true,
  };
}

/**
 * Create a system message (for errors, info, etc.).
 */
export function createSystemMessage(content: string): AIMessage {
  return {
    id: `system-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
    role: 'system',
    content,
    timestamp: new Date(),
  };
}

// ============================================================================
// Sidebar Dimension Constants
// ============================================================================

/**
 * Minimum sidebar width (increased from 240px for chat readability).
 */
export const SIDEBAR_MIN_WIDTH = 320;

/**
 * Maximum sidebar width.
 */
export const SIDEBAR_MAX_WIDTH = 640;

/**
 * Default sidebar width.
 */
export const SIDEBAR_DEFAULT_WIDTH = 360;

// ============================================================================
// LocalStorage Keys
// ============================================================================

export const STORAGE_KEYS = {
  SIDEBAR_WIDTH: 'unified-sidebar-width',
  SIDEBAR_OPEN: 'unified-sidebar-open',
  SIDEBAR_TAB: 'unified-sidebar-tab',
  AI_MODEL: 'ai-navigation-model',
  AI_MAX_STEPS: 'ai-navigation-max-steps',
  AI_INPUT_DRAFT: 'ai-navigation-input-draft',
} as const;

// ============================================================================
// Re-exports from ai-navigation for convenience
// ============================================================================

export type { AINavigationStep, HumanInterventionState, TokenUsage };
