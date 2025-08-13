/**
 * TypeScript Type Definitions for {{APP_NAME}} Extension
 * 
 * AGENT NOTE: Add app-specific types here as needed.
 * This file helps maintain type safety across the extension.
 */

// ============================================
// Chrome Extension API Types (for missing ones)
// ============================================

// Chrome already provides most types via @types/chrome
// Add any custom extension types here if needed

// ============================================
// Message Types (for communication between components)
// ============================================

export type MessageType = 
    | 'AUTH_CHECK'
    | 'LOGIN'
    | 'LOGOUT'
    | 'GET_CURRENT_TAB_INFO'
    | '{{PRIMARY_ACTION_MESSAGE}}'
    // Add more message types as needed
    ;

export interface Message {
    type: MessageType;
    data?: any;
    credentials?: {
        email: string;
        password: string;
    };
}

export interface MessageResponse {
    success?: boolean;
    error?: string;
    data?: any;
    isAuthenticated?: boolean;
    user?: User;
    tab?: chrome.tabs.Tab;
}

// ============================================
// Data Models
// ============================================

export interface User {
    id: string;
    name: string;
    email: string;
    avatar?: string;
    // Add app-specific user fields
}

export interface {{AppModel}} {
    id: string;
    title: string;
    description?: string;
    createdAt: string;
    updatedAt: string;
    // Add app-specific model fields
}

// ============================================
// Storage Types
// ============================================

export interface StorageData {
    authToken?: string;
    extensionState?: ExtensionState;
    userPreferences?: UserPreferences;
    cachedData?: CachedData;
}

export interface ExtensionState {
    isAuthenticated: boolean;
    user: User | null;
    lastSync: number;
    // Add app-specific state
}

export interface UserPreferences {
    theme?: 'light' | 'dark' | 'auto';
    notifications?: boolean;
    autoSync?: boolean;
    // Add app-specific preferences
}

export interface CachedData {
    items?: {{AppModel}}[];
    lastFetch?: number;
    // Add app-specific cached data
}

// ============================================
// API Types
// ============================================

export interface ApiRequest {
    endpoint: string;
    method?: 'GET' | 'POST' | 'PUT' | 'DELETE';
    body?: any;
    headers?: Record<string, string>;
}

export interface ApiResponse<T = any> {
    success: boolean;
    data?: T;
    error?: string;
    message?: string;
}

export interface LoginResponse {
    token: string;
    user: User;
    expiresAt?: string;
}

// ============================================
// Content Script Types (if using content scripts)
// ============================================

export interface ContentScriptMessage {
    action: 'EXTRACT_DATA' | 'HIGHLIGHT_ELEMENT' | 'INJECT_UI';
    data?: any;
}

export interface PageData {
    url: string;
    title: string;
    description?: string;
    images?: string[];
    // Add data that can be extracted from pages
}

// ============================================
// Configuration Types
// ============================================

export interface ExtensionConfig {
    API_BASE: string;
    APP_DOMAIN: string;
    SYNC_INTERVAL: number;
    MAX_RETRY_ATTEMPTS: number;
    // Add app-specific configuration
}

// ============================================
// Utility Types
// ============================================

export type AsyncFunction<T = void> = () => Promise<T>;

export type ErrorCallback = (error: Error) => void;

export type SuccessCallback<T = any> = (data: T) => void;

// Generic result type for operations that can fail
export type Result<T, E = Error> = 
    | { success: true; data: T }
    | { success: false; error: E };

// ============================================
// Event Types
// ============================================

export interface ExtensionEvent {
    type: string;
    timestamp: number;
    data?: any;
}

export interface TabEvent extends ExtensionEvent {
    tabId: number;
    url?: string;
    status?: string;
}

// ============================================
// Notification Types
// ============================================

export interface NotificationOptions {
    title: string;
    message: string;
    iconUrl?: string;
    type?: 'basic' | 'image' | 'list' | 'progress';
    priority?: 0 | 1 | 2;
    buttons?: Array<{ title: string }>;
}

// ============================================
// Context Menu Types
// ============================================

export interface ContextMenuData {
    menuItemId: string;
    selectionText?: string;
    linkUrl?: string;
    srcUrl?: string;
    pageUrl: string;
}

// ============================================
// Badge Types
// ============================================

export interface BadgeConfig {
    text: string;
    backgroundColor?: string;
    textColor?: string;
    tabId?: number;
}

// ============================================
// Export all types for easy importing
// ============================================

export * from './types';