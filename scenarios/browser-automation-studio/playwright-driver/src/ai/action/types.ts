/**
 * Browser Action Types
 *
 * STABILITY: STABLE CONTRACT
 *
 * This module defines the union type for all browser actions that the vision agent
 * can execute. Each action maps to a Playwright operation.
 *
 * Changes here affect the action parser, executor, and vision model prompts.
 * Add new action types with care - they require parser/executor updates.
 */

/**
 * Union type for all browser actions.
 * Each action maps to a Playwright operation.
 */
export type BrowserAction =
  | ClickAction
  | TypeAction
  | ScrollAction
  | NavigateAction
  | HoverAction
  | SelectAction
  | WaitAction
  | KeyPressAction
  | DoneAction
  | RequestHumanAction;

/**
 * Click on an element or coordinates.
 * Uses element ID from annotated screenshot when available.
 */
export interface ClickAction {
  type: 'click';
  /** Element label number from annotated screenshot */
  elementId?: number;
  /** Pixel coordinates (fallback if no elementId) */
  coordinates?: { x: number; y: number };
  /** Click variant */
  variant?: 'left' | 'right' | 'double';
}

/**
 * Type text into an input field.
 * Can target specific element or use currently focused element.
 */
export interface TypeAction {
  type: 'type';
  /** Text to type */
  text: string;
  /** Element label number (optional - uses focused element if omitted) */
  elementId?: number;
  /** Whether to clear existing text first */
  clearFirst?: boolean;
}

/**
 * Scroll the page in a direction.
 */
export interface ScrollAction {
  type: 'scroll';
  direction: 'up' | 'down' | 'left' | 'right';
  /** Scroll amount in pixels (default: viewport height/width) */
  amount?: number;
}

/**
 * Navigate to a URL.
 */
export interface NavigateAction {
  type: 'navigate';
  url: string;
}

/**
 * Hover over an element.
 */
export interface HoverAction {
  type: 'hover';
  elementId?: number;
  coordinates?: { x: number; y: number };
}

/**
 * Select an option from a dropdown.
 */
export interface SelectAction {
  type: 'select';
  elementId: number;
  value: string;
}

/**
 * Wait for a condition.
 */
export interface WaitAction {
  type: 'wait';
  /** Milliseconds to wait */
  ms?: number;
  /** Selector to wait for */
  selector?: string;
}

/**
 * Press a keyboard key or combination.
 */
export interface KeyPressAction {
  type: 'keypress';
  key: string; // e.g., "Enter", "Escape", "Tab"
  modifiers?: ('Control' | 'Alt' | 'Shift' | 'Meta')[];
}

/**
 * Signal that the task is complete.
 */
export interface DoneAction {
  type: 'done';
  /** Summary of what was accomplished */
  result: string;
  /** Whether the goal was successfully achieved */
  success: boolean;
}

/**
 * Request human intervention for tasks the AI cannot complete.
 * Used when encountering CAPTCHAs, human verification, or complex interactive elements.
 */
export interface RequestHumanAction {
  type: 'request_human';
  /** Why human intervention is needed */
  reason: string;
  /** Instructions for the human user */
  instructions?: string;
  /** Type of intervention needed */
  interventionType: 'captcha' | 'verification' | 'complex_interaction' | 'login_required' | 'other';
}

/**
 * Extract the action type as a string literal type.
 * Useful for switch statements and type guards.
 */
export type ActionType = BrowserAction['type'];

/**
 * Type guard to check if an action is a click action.
 */
export function isClickAction(action: BrowserAction): action is ClickAction {
  return action.type === 'click';
}

/**
 * Type guard to check if an action is a type action.
 */
export function isTypeAction(action: BrowserAction): action is TypeAction {
  return action.type === 'type';
}

/**
 * Type guard to check if an action is a done action.
 */
export function isDoneAction(action: BrowserAction): action is DoneAction {
  return action.type === 'done';
}

/**
 * Type guard to check if an action is a request human action.
 */
export function isRequestHumanAction(action: BrowserAction): action is RequestHumanAction {
  return action.type === 'request_human';
}
