const { contextBridge, ipcRenderer } = require('electron');

// Expose protected APIs to the renderer
contextBridge.exposeInMainWorld('electronAPI', {
  // Screenshot capture
  captureScreenshot: () => ipcRenderer.invoke('capture-screenshot'),
  
  // Context gathering
  getContext: () => ipcRenderer.invoke('get-context'),
  
  // Issue submission
  submitIssue: (data) => ipcRenderer.invoke('submit-issue', data),
  
  // Agent spawning
  spawnAgent: (issueData, agentType) => ipcRenderer.invoke('spawn-agent', issueData, agentType),
  
  // Window control
  hideOverlay: () => ipcRenderer.invoke('hide-overlay'),
  
  // Settings
  getSettings: () => ipcRenderer.invoke('get-settings'),
  saveSettings: (settings) => ipcRenderer.invoke('save-settings', settings),
  
  // History
  getHistory: () => ipcRenderer.invoke('get-history'),
  
  // Clipboard
  copyToClipboard: (text) => {
    navigator.clipboard.writeText(text);
  },
  
  // File system (limited)
  readFile: (path) => ipcRenderer.invoke('read-file', path),
  
  // Platform info
  platform: process.platform,
  version: process.versions.electron
});