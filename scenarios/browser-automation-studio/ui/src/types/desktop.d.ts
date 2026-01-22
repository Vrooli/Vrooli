/**
 * Type declarations for the Electron desktop API.
 * These are provided by the preload script when running in Electron.
 */

interface DesktopAuthAPI {
  signIn: () => Promise<{ state: string }>;
  signOut: () => Promise<void>;
  getAccessToken: () => Promise<string | null>;
  getUser: () => Promise<{
    id: string;
    email: string;
    emailVerified: boolean;
  } | null>;
  isAuthenticated: () => Promise<boolean>;
  refresh: () => Promise<boolean>;
  onAuthChanged: (callback: (event: 'tokens-received' | 'tokens-refreshed' | 'session-expired' | 'signed-out') => void) => void;
  offAuthChanged: (callback: (event: string) => void) => void;
}

interface DesktopTrayAPI {
  updateTooltip: (tooltip: string) => Promise<{ success: boolean }>;
  setBadge: (count: number) => Promise<{ success: boolean }>;
  updateContextMenu: (items: { label: string; action: string; enabled?: boolean }[]) => Promise<{ success: boolean }>;
}

interface DesktopStorageAPI {
  getStoragePath: () => Promise<string>;
  ensureDir: (relativePath: string) => Promise<void>;
  writeFile: (relativePath: string, data: string | ArrayBuffer) => Promise<void>;
  readFile: (relativePath: string) => Promise<ArrayBuffer | null>;
  readTextFile: (relativePath: string) => Promise<string | null>;
  deleteFile: (relativePath: string) => Promise<boolean>;
  deleteDir: (relativePath: string) => Promise<boolean>;
  listDir: (relativePath: string) => Promise<{
    name: string;
    path: string;
    isDirectory: boolean;
    isFile: boolean;
    size: number;
    createdAt: number;
    modifiedAt: number;
  }[] | null>;
  exists: (relativePath: string) => Promise<boolean>;
  stat: (relativePath: string) => Promise<{
    size: number;
    createdAt: number;
    modifiedAt: number;
    isDirectory: boolean;
    isFile: boolean;
  } | null>;
  getStorageInfo: () => Promise<{
    used: number;
    available: number;
    total?: number; // Alias for available (backwards compat)
    count: number;
  }>;
}

interface DesktopAPI {
  // File operations (dialog-based)
  save: (content: string, defaultPath?: string) => Promise<string | null>;
  open: () => Promise<{ filePath: string; content: string } | null>;
  saveJSON: (data: unknown, defaultFilename?: string) => Promise<string | null>;
  loadJSON: () => Promise<{ filePath: string; data: unknown } | null>;

  // App-managed storage
  storage: DesktopStorageAPI;
  storeJSON: (relativePath: string, data: unknown) => Promise<void>;
  loadStoredJSON: (relativePath: string) => Promise<unknown | null>;
  storeBlob: (relativePath: string, data: ArrayBuffer | Blob) => Promise<void>;
  loadStoredBlob: (relativePath: string, mimeType?: string) => Promise<Blob | null>;
  getStoredFileUrl: (relativePath: string, mimeType?: string) => Promise<string | null>;

  // Authentication
  auth: DesktopAuthAPI;

  // System tray
  tray?: DesktopTrayAPI;

  // App control & utilities
  notify: (title: string, body: string, options?: { silent?: boolean; urgency?: 'low' | 'normal' | 'critical' }) => void;
  minimize: () => Promise<void>;
  maximize: () => Promise<void>;
  close: () => Promise<void>;
  getInfo: () => Promise<{
    platform: string;
    arch: string;
    version: string;
    appVersion: string;
    isPackaged: boolean;
  }>;
  onMenuAction: (callback: (action: string, data?: unknown) => void) => void;
  onProtocolUrl: (callback: (url: string) => void) => void;

  // Feature flags
  features: {
    fileSystem: boolean;
    appStorage: boolean;
    notifications: boolean;
    auth: boolean;
    systemTray: boolean;
    autoUpdater: boolean;
    multiWindow: boolean;
  };
}

declare global {
  interface Window {
    desktop?: DesktopAPI;
    desktopAPI?: unknown;
    desktopUtils?: unknown;
  }
}

export {};
