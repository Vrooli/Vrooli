/**
 * File Operations Example for Electron Desktop Apps
 *
 * This file demonstrates how to add file save/open functionality to your desktop app.
 *
 * USAGE:
 * 1. Copy the preload code to your preload.ts file
 * 2. Copy the main process code to your main.ts file
 * 3. Use the React component as a template in your UI
 */

// ============================================================================
// STEP 1: Add to preload.ts
// ============================================================================

import { contextBridge, ipcRenderer } from 'electron';

contextBridge.exposeInMainWorld('electronAPI', {
  // Save file with dialog
  saveFile: (filename: string, content: string) =>
    ipcRenderer.invoke('save-file', filename, content),

  // Open file with dialog
  openFile: () =>
    ipcRenderer.invoke('open-file'),

  // Save file to specific path (no dialog)
  saveFileTo: (filePath: string, content: string) =>
    ipcRenderer.invoke('save-file-to', filePath, content),

  // Read file from specific path
  readFileFrom: (filePath: string) =>
    ipcRenderer.invoke('read-file-from', filePath),

  // Select directory
  selectDirectory: () =>
    ipcRenderer.invoke('select-directory'),

  // Select multiple files
  selectFiles: (filters?: { name: string; extensions: string[] }[]) =>
    ipcRenderer.invoke('select-files', filters),
});

// ============================================================================
// STEP 2: Add to main.ts
// ============================================================================

import { app, dialog, ipcMain } from 'electron';
import fs from 'fs/promises';
import path from 'path';

// Save file with dialog
ipcMain.handle('save-file', async (event, filename, content) => {
  const { filePath } = await dialog.showSaveDialog({
    defaultPath: filename,
    filters: [
      { name: 'JSON Files', extensions: ['json'] },
      { name: 'Text Files', extensions: ['txt'] },
      { name: 'All Files', extensions: ['*'] }
    ]
  });

  if (filePath) {
    try {
      await fs.writeFile(filePath, content, 'utf-8');
      return { success: true, path: filePath };
    } catch (error) {
      return { success: false, error: (error as Error).message };
    }
  }
  return { success: false, error: 'User cancelled' };
});

// Open file with dialog
ipcMain.handle('open-file', async () => {
  const { filePaths } = await dialog.showOpenDialog({
    properties: ['openFile'],
    filters: [
      { name: 'JSON Files', extensions: ['json'] },
      { name: 'Text Files', extensions: ['txt'] },
      { name: 'All Files', extensions: ['*'] }
    ]
  });

  if (filePaths.length > 0) {
    try {
      const content = await fs.readFile(filePaths[0], 'utf-8');
      return {
        success: true,
        content,
        path: filePaths[0],
        name: path.basename(filePaths[0])
      };
    } catch (error) {
      return { success: false, error: (error as Error).message };
    }
  }
  return { success: false, error: 'User cancelled' };
});

// Save file to specific path (no dialog)
ipcMain.handle('save-file-to', async (event, filePath, content) => {
  try {
    await fs.writeFile(filePath, content, 'utf-8');
    return { success: true, path: filePath };
  } catch (error) {
    return { success: false, error: (error as Error).message };
  }
});

// Read file from specific path
ipcMain.handle('read-file-from', async (event, filePath) => {
  try {
    const content = await fs.readFile(filePath, 'utf-8');
    return { success: true, content, path: filePath };
  } catch (error) {
    return { success: false, error: (error as Error).message };
  }
});

// Select directory
ipcMain.handle('select-directory', async () => {
  const { filePaths } = await dialog.showOpenDialog({
    properties: ['openDirectory']
  });

  if (filePaths.length > 0) {
    return { success: true, path: filePaths[0] };
  }
  return { success: false, error: 'User cancelled' };
});

// Select multiple files
ipcMain.handle('select-files', async (event, filters) => {
  const { filePaths } = await dialog.showOpenDialog({
    properties: ['openFile', 'multiSelections'],
    filters: filters || [{ name: 'All Files', extensions: ['*'] }]
  });

  if (filePaths.length > 0) {
    return { success: true, paths: filePaths };
  }
  return { success: false, error: 'User cancelled' };
});

// ============================================================================
// STEP 3: TypeScript Declarations (add to your .d.ts file or component)
// ============================================================================

declare global {
  interface Window {
    electronAPI: {
      saveFile: (filename: string, content: string) => Promise<{
        success: boolean;
        path?: string;
        error?: string;
      }>;
      openFile: () => Promise<{
        success: boolean;
        content?: string;
        path?: string;
        name?: string;
        error?: string;
      }>;
      saveFileTo: (filePath: string, content: string) => Promise<{
        success: boolean;
        path?: string;
        error?: string;
      }>;
      readFileFrom: (filePath: string) => Promise<{
        success: boolean;
        content?: string;
        path?: string;
        error?: string;
      }>;
      selectDirectory: () => Promise<{
        success: boolean;
        path?: string;
        error?: string;
      }>;
      selectFiles: (filters?: { name: string; extensions: string[] }[]) => Promise<{
        success: boolean;
        paths?: string[];
        error?: string;
      }>;
    };
  }
}

// ============================================================================
// STEP 4: Example React Components
// ============================================================================

import React, { useState } from 'react';

// Example 1: Save JSON data
export function SaveButton({ data }: { data: any }) {
  const handleSave = async () => {
    if (!window.electronAPI) {
      // Fallback for web - download as file
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'export.json';
      a.click();
      return;
    }

    // Desktop - use native dialog
    const result = await window.electronAPI.saveFile(
      'export.json',
      JSON.stringify(data, null, 2)
    );

    if (result.success) {
      alert(`File saved to ${result.path}`);
    } else {
      alert(`Failed to save: ${result.error}`);
    }
  };

  return (
    <button onClick={handleSave}>
      Save Data
    </button>
  );
}

// Example 2: Load JSON data
export function LoadButton({ onLoad }: { onLoad: (data: any) => void }) {
  const handleLoad = async () => {
    if (!window.electronAPI) {
      // Fallback for web - file input
      const input = document.createElement('input');
      input.type = 'file';
      input.accept = '.json';
      input.onchange = async (e) => {
        const file = (e.target as HTMLInputElement).files?.[0];
        if (file) {
          const text = await file.text();
          onLoad(JSON.parse(text));
        }
      };
      input.click();
      return;
    }

    // Desktop - use native dialog
    const result = await window.electronAPI.openFile();

    if (result.success && result.content) {
      try {
        const data = JSON.parse(result.content);
        onLoad(data);
      } catch (error) {
        alert('Invalid JSON file');
      }
    } else if (result.error && result.error !== 'User cancelled') {
      alert(`Failed to load: ${result.error}`);
    }
  };

  return (
    <button onClick={handleLoad}>
      Load Data
    </button>
  );
}

// Example 3: Export to directory
export function ExportToFolderButton({ data }: { data: any }) {
  const [exportPath, setExportPath] = useState<string>('');

  const handleSelectFolder = async () => {
    if (!window.electronAPI) {
      alert('This feature is only available in the desktop app');
      return;
    }

    const result = await window.electronAPI.selectDirectory();
    if (result.success && result.path) {
      setExportPath(result.path);
    }
  };

  const handleExport = async () => {
    if (!exportPath || !window.electronAPI) return;

    const filePath = `${exportPath}/export-${Date.now()}.json`;
    const result = await window.electronAPI.saveFileTo(
      filePath,
      JSON.stringify(data, null, 2)
    );

    if (result.success) {
      alert(`Exported to ${result.path}`);
    } else {
      alert(`Failed to export: ${result.error}`);
    }
  };

  if (!window.electronAPI) {
    return null; // Hide on web
  }

  return (
    <div>
      <button onClick={handleSelectFolder}>
        Select Export Folder
      </button>
      {exportPath && (
        <>
          <p>Export to: {exportPath}</p>
          <button onClick={handleExport}>
            Export Now
          </button>
        </>
      )}
    </div>
  );
}

// ============================================================================
// USAGE TIPS
// ============================================================================

/**
 * 1. Graceful Degradation:
 *    Always check if window.electronAPI exists before using it.
 *    Provide web fallbacks using browser APIs.
 *
 * 2. Error Handling:
 *    Always handle both success and error cases.
 *    Show user-friendly error messages.
 *
 * 3. Security:
 *    Never allow arbitrary file system access.
 *    Always validate file paths and content.
 *    Use dialog.showOpenDialog/showSaveDialog for user selection.
 *
 * 4. File Types:
 *    Customize the filters array for your specific file types.
 *    Example: [{ name: 'Images', extensions: ['png', 'jpg', 'gif'] }]
 *
 * 5. Large Files:
 *    For large files, consider streaming or chunked reading.
 *    Show progress indicators for long operations.
 */
