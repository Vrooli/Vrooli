/**
 * Sidebar Domain Types
 *
 * Types for the unified sidebar with Timeline, Auto, and Artifacts tabs.
 */

import type { AINavigationStep, HumanInterventionState, TokenUsage } from '../ai-navigation/types';
import type { TimelineMode } from '../types/timeline-unified';

// ============================================================================
// Tab State Types
// ============================================================================

/**
 * Available sidebar tabs.
 * - timeline: Recording/execution timeline (both modes)
 * - auto: AI navigation chat (recording mode only)
 * - artifacts: Execution artifacts including screenshots, logs, network, DOM (execution mode only)
 * - history: Execution history for switching between runs (execution mode only)
 */
export type TabId = 'timeline' | 'auto' | 'artifacts' | 'history';

/**
 * Sub-types within the Artifacts tab.
 * Allows switching between different artifact views.
 */
export type ArtifactSubType =
  | 'screenshots'        // Captured screenshots
  | 'execution-logs'     // Execution progress logs (step started/completed/failed)
  | 'console'            // Browser console logs
  | 'network'            // Network requests/responses
  | 'dom-snapshots';     // DOM snapshots

/**
 * Configuration for a sidebar tab.
 */
export interface TabConfig {
  /** Unique tab identifier */
  id: TabId;
  /** Display label */
  label: string;
  /** Tooltip description */
  tooltip: string;
  /** Which modes this tab is visible in */
  visibleIn: TimelineMode[];
  /** Keyboard shortcut number (1-9) - assigned dynamically based on visible tabs */
  shortcutKey?: number;
}

/**
 * Tab configurations with visibility rules.
 * Note: shortcutKey is assigned dynamically based on visible tabs order.
 */
export const TAB_CONFIGS: TabConfig[] = [
  {
    id: 'timeline',
    label: 'Timeline',
    tooltip: 'View recorded or executed actions',
    visibleIn: ['recording', 'execution'],
  },
  {
    id: 'auto',
    label: 'Auto',
    tooltip: 'AI-powered browser automation',
    visibleIn: ['recording'],
  },
  {
    id: 'artifacts',
    label: 'Artifacts',
    tooltip: 'Screenshots, logs, network requests, and more',
    visibleIn: ['execution'],
  },
  {
    id: 'history',
    label: 'History',
    tooltip: 'Execution history for this workflow',
    visibleIn: ['execution'],
  },
];

/**
 * Default artifact sub-type.
 */
export const DEFAULT_ARTIFACT_SUBTYPE: ArtifactSubType = 'screenshots';

/**
 * Get visible tabs for a given mode.
 */
export function getVisibleTabs(mode: TimelineMode): TabConfig[] {
  return TAB_CONFIGS.filter(tab => tab.visibleIn.includes(mode)).map((tab, index) => ({
    ...tab,
    shortcutKey: index + 1,
  }));
}

/**
 * Check if a tab is visible in a given mode.
 */
export function isTabVisible(tabId: TabId, mode: TimelineMode): boolean {
  const tab = TAB_CONFIGS.find(t => t.id === tabId);
  return tab ? tab.visibleIn.includes(mode) : false;
}

/**
 * Get default tab for a mode.
 */
export function getDefaultTab(mode: TimelineMode): TabId {
  const visibleTabs = getVisibleTabs(mode);
  return visibleTabs.length > 0 ? visibleTabs[0].id : 'timeline';
}

/**
 * State for the unified sidebar.
 */
export interface UnifiedSidebarState {
  activeTab: TabId;
  width: number;
  isResizing: boolean;
  timelineActivity: boolean;
  autoActivity: boolean;
  artifactsActivity: boolean;
  historyActivity: boolean;
  /** Current sub-type selected within Artifacts tab */
  artifactSubType: ArtifactSubType;
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
export type AIMessageStatus = 'pending' | 'running' | 'aborting' | 'completed' | 'failed' | 'aborted' | 'awaiting_human';

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
  ARTIFACT_SUBTYPE: 'unified-sidebar-artifact-subtype',
  AI_MODEL: 'ai-navigation-model',
  AI_MAX_STEPS: 'ai-navigation-max-steps',
  AI_INPUT_DRAFT: 'ai-navigation-input-draft',
} as const;

// ============================================================================
// Re-exports from ai-navigation for convenience
// ============================================================================

export type { AINavigationStep, HumanInterventionState, TokenUsage };
