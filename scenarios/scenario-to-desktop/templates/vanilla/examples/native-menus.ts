/**
 * Native Menus Example for Electron Desktop Apps
 *
 * This file demonstrates how to add native application menus with keyboard shortcuts.
 *
 * USAGE:
 * 1. Copy the menu creation code to your main.ts file
 * 2. Copy the IPC handlers to communicate with React
 * 3. Add event listeners in your React components
 */

// ============================================================================
// STEP 1: Add to main.ts - Create Application Menu
// ============================================================================

import { app, Menu, MenuItem, BrowserWindow } from 'electron';

let mainWindow: BrowserWindow;

function createApplicationMenu() {
  const isMac = process.platform === 'darwin';

  const template: (MenuItem | MenuItemConstructorOptions)[] = [
    // App menu (macOS only)
    ...(isMac ? [{
      label: app.name,
      submenu: [
        { role: 'about' as const },
        { type: 'separator' as const },
        { role: 'services' as const },
        { type: 'separator' as const },
        { role: 'hide' as const },
        { role: 'hideOthers' as const },
        { role: 'unhide' as const },
        { type: 'separator' as const },
        { role: 'quit' as const }
      ]
    }] : []),

    // File menu
    {
      label: 'File',
      submenu: [
        {
          label: 'New',
          accelerator: 'CmdOrCtrl+N',
          click: () => {
            mainWindow.webContents.send('menu-new');
          }
        },
        {
          label: 'Open...',
          accelerator: 'CmdOrCtrl+O',
          click: () => {
            mainWindow.webContents.send('menu-open');
          }
        },
        {
          label: 'Save',
          accelerator: 'CmdOrCtrl+S',
          click: () => {
            mainWindow.webContents.send('menu-save');
          }
        },
        {
          label: 'Save As...',
          accelerator: 'CmdOrCtrl+Shift+S',
          click: () => {
            mainWindow.webContents.send('menu-save-as');
          }
        },
        { type: 'separator' as const },
        {
          label: 'Export...',
          accelerator: 'CmdOrCtrl+E',
          click: () => {
            mainWindow.webContents.send('menu-export');
          }
        },
        { type: 'separator' as const },
        isMac ? { role: 'close' as const } : { role: 'quit' as const }
      ]
    },

    // Edit menu
    {
      label: 'Edit',
      submenu: [
        { role: 'undo' as const },
        { role: 'redo' as const },
        { type: 'separator' as const },
        { role: 'cut' as const },
        { role: 'copy' as const },
        { role: 'paste' as const },
        ...(isMac ? [
          { role: 'pasteAndMatchStyle' as const },
          { role: 'delete' as const },
          { role: 'selectAll' as const },
          { type: 'separator' as const },
          {
            label: 'Speech',
            submenu: [
              { role: 'startSpeaking' as const },
              { role: 'stopSpeaking' as const }
            ]
          }
        ] : [
          { role: 'delete' as const },
          { type: 'separator' as const },
          { role: 'selectAll' as const }
        ])
      ]
    },

    // View menu
    {
      label: 'View',
      submenu: [
        { role: 'reload' as const },
        { role: 'forceReload' as const },
        { role: 'toggleDevTools' as const },
        { type: 'separator' as const },
        { role: 'resetZoom' as const },
        { role: 'zoomIn' as const },
        { role: 'zoomOut' as const },
        { type: 'separator' as const },
        { role: 'togglefullscreen' as const }
      ]
    },

    // Window menu
    {
      label: 'Window',
      submenu: [
        { role: 'minimize' as const },
        { role: 'zoom' as const },
        ...(isMac ? [
          { type: 'separator' as const },
          { role: 'front' as const },
          { type: 'separator' as const },
          { role: 'window' as const }
        ] : [
          { role: 'close' as const }
        ])
      ]
    },

    // Help menu
    {
      role: 'help' as const,
      submenu: [
        {
          label: 'Learn More',
          click: async () => {
            const { shell } = require('electron');
            await shell.openExternal('https://github.com/Vrooli/Vrooli');
          }
        },
        {
          label: 'Documentation',
          click: async () => {
            const { shell } = require('electron');
            await shell.openExternal('https://docs.vrooli.com');
          }
        },
        { type: 'separator' as const },
        {
          label: 'About',
          click: () => {
            mainWindow.webContents.send('menu-about');
          }
        }
      ]
    }
  ];

  const menu = Menu.buildFromTemplate(template);
  Menu.setApplicationMenu(menu);
}

// Call this after creating your main window
app.whenReady().then(() => {
  // ... create mainWindow ...
  createApplicationMenu();
});

// ============================================================================
// STEP 2: Add to preload.ts - Menu Event Listeners
// ============================================================================

import { contextBridge, ipcRenderer } from 'electron';

contextBridge.exposeInMainWorld('electronAPI', {
  // Listen for menu events
  onMenuNew: (callback: () => void) => {
    ipcRenderer.on('menu-new', callback);
  },
  onMenuOpen: (callback: () => void) => {
    ipcRenderer.on('menu-open', callback);
  },
  onMenuSave: (callback: () => void) => {
    ipcRenderer.on('menu-save', callback);
  },
  onMenuSaveAs: (callback: () => void) => {
    ipcRenderer.on('menu-save-as', callback);
  },
  onMenuExport: (callback: () => void) => {
    ipcRenderer.on('menu-export', callback);
  },
  onMenuAbout: (callback: () => void) => {
    ipcRenderer.on('menu-about', callback);
  },

  // Remove listeners (for cleanup)
  removeMenuListeners: () => {
    ipcRenderer.removeAllListeners('menu-new');
    ipcRenderer.removeAllListeners('menu-open');
    ipcRenderer.removeAllListeners('menu-save');
    ipcRenderer.removeAllListeners('menu-save-as');
    ipcRenderer.removeAllListeners('menu-export');
    ipcRenderer.removeAllListeners('menu-about');
  }
});

// ============================================================================
// STEP 3: TypeScript Declarations
// ============================================================================

declare global {
  interface Window {
    electronAPI: {
      onMenuNew: (callback: () => void) => void;
      onMenuOpen: (callback: () => void) => void;
      onMenuSave: (callback: () => void) => void;
      onMenuSaveAs: (callback: () => void) => void;
      onMenuExport: (callback: () => void) => void;
      onMenuAbout: (callback: () => void) => void;
      removeMenuListeners: () => void;
    };
  }
}

// ============================================================================
// STEP 4: React Component Example
// ============================================================================

import React, { useEffect } from 'react';

export function AppWithMenus() {
  useEffect(() => {
    if (!window.electronAPI) return; // Web mode - no menus

    // Register menu handlers
    window.electronAPI.onMenuNew(() => {
      console.log('New menu item clicked');
      // Handle new document/project
    });

    window.electronAPI.onMenuOpen(() => {
      console.log('Open menu item clicked');
      // Trigger file open dialog
    });

    window.electronAPI.onMenuSave(() => {
      console.log('Save menu item clicked');
      // Save current document
    });

    window.electronAPI.onMenuSaveAs(() => {
      console.log('Save As menu item clicked');
      // Show save dialog
    });

    window.electronAPI.onMenuExport(() => {
      console.log('Export menu item clicked');
      // Export data
    });

    window.electronAPI.onMenuAbout(() => {
      console.log('About menu item clicked');
      // Show about dialog
    });

    // Cleanup on unmount
    return () => {
      window.electronAPI?.removeMenuListeners();
    };
  }, []);

  return <div>Your app content</div>;
}

// ============================================================================
// STEP 5: Advanced - Dynamic Menus
// ============================================================================

// In main.ts - add IPC handler to update menus dynamically

import { ipcMain } from 'electron';

ipcMain.handle('update-menu-state', (event, menuState) => {
  const menu = Menu.getApplicationMenu();
  if (!menu) return;

  // Enable/disable menu items based on app state
  const fileMenu = menu.items.find(item => item.label === 'File');
  if (fileMenu && fileMenu.submenu) {
    const saveItem = fileMenu.submenu.items.find(item => item.label === 'Save');
    if (saveItem) {
      saveItem.enabled = menuState.canSave;
    }

    const exportItem = fileMenu.submenu.items.find(item => item.label === 'Export...');
    if (exportItem) {
      exportItem.enabled = menuState.canExport;
    }
  }
});

// In preload.ts - add to electronAPI

contextBridge.exposeInMainWorld('electronAPI', {
  // ... other methods ...
  updateMenuState: (state: { canSave: boolean; canExport: boolean }) =>
    ipcRenderer.invoke('update-menu-state', state),
});

// In React component - update menu state

useEffect(() => {
  if (window.electronAPI) {
    window.electronAPI.updateMenuState({
      canSave: hasUnsavedChanges,
      canExport: hasData
    });
  }
}, [hasUnsavedChanges, hasData]);

// ============================================================================
// STEP 6: Custom Context Menus
// ============================================================================

// In main.ts - add right-click context menu

import { Menu } from 'electron';

mainWindow.webContents.on('context-menu', (event, params) => {
  const template: MenuItemConstructorOptions[] = [];

  // Add spell check suggestions if text is misspelled
  if (params.misspelledWord) {
    for (const suggestion of params.dictionarySuggestions) {
      template.push({
        label: suggestion,
        click: () => {
          mainWindow.webContents.replaceMisspelling(suggestion);
        }
      });
    }
    template.push({ type: 'separator' });
  }

  // Add standard editing options
  if (params.isEditable) {
    template.push(
      { role: 'undo' },
      { role: 'redo' },
      { type: 'separator' },
      { role: 'cut' },
      { role: 'copy' },
      { role: 'paste' },
      { type: 'separator' },
      { role: 'selectAll' }
    );
  }

  // Add link options
  if (params.linkURL) {
    template.push({
      label: 'Open Link',
      click: () => {
        require('electron').shell.openExternal(params.linkURL);
      }
    });
  }

  // Add image options
  if (params.mediaType === 'image') {
    template.push({
      label: 'Save Image',
      click: () => {
        // Download image logic
      }
    });
  }

  if (template.length > 0) {
    const menu = Menu.buildFromTemplate(template);
    menu.popup();
  }
});

// ============================================================================
// USAGE TIPS
// ============================================================================

/**
 * 1. Platform Differences:
 *    - macOS has app menu, Windows/Linux don't
 *    - Use process.platform to detect OS
 *    - Test on all platforms
 *
 * 2. Keyboard Shortcuts:
 *    - CmdOrCtrl auto-adapts to Cmd (Mac) or Ctrl (Win/Linux)
 *    - Common shortcuts: Ctrl+N, Ctrl+O, Ctrl+S, etc.
 *    - Avoid conflicting with system shortcuts
 *
 * 3. Menu Roles:
 *    - Use built-in roles when possible (undo, redo, cut, copy, paste)
 *    - Electron handles these automatically
 *    - Provides consistent behavior across platforms
 *
 * 4. Dynamic Menus:
 *    - Enable/disable items based on app state
 *    - Update menu labels dynamically
 *    - Use ipcMain.handle for state updates
 *
 * 5. Accessibility:
 *    - Always provide keyboard shortcuts
 *    - Use clear, descriptive labels
 *    - Follow platform conventions
 */
