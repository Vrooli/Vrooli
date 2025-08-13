/**
 * ELECTRON PRELOAD SCRIPT TEMPLATE
 * 
 * This script bridges the gap between your web app and Electron's native capabilities.
 * It runs in an isolated context with access to both web APIs and a limited set of Node.js APIs.
 * 
 * AI AGENT INSTRUCTIONS:
 * 1. Uncomment and modify the APIs your app needs
 * 2. Add app-specific IPC channels for communication with main process
 * 3. Ensure all exposed APIs follow security best practices
 * 
 * Learn more: https://www.electronjs.org/docs/latest/tutorial/process-model#preload-scripts
 */

const { contextBridge, ipcRenderer } = require("electron");

// ===== EXPOSE NATIVE APIS TO WEB APP =====
// AI AGENT NOTE: Customize this based on what native features your app needs
contextBridge.exposeInMainWorld("electronAPI", {
  
  // === File System Operations ===
  openFile: () => ipcRenderer.invoke("dialog:openFile"),
  saveFile: (data) => ipcRenderer.invoke("dialog:saveFile", data),
  readFile: (path) => ipcRenderer.invoke("fs:readFile", path),
  writeFile: (path, data) => ipcRenderer.invoke("fs:writeFile", path, data),
  
  // === Application Control ===
  minimize: () => ipcRenderer.send("window:minimize"),
  maximize: () => ipcRenderer.send("window:maximize"),
  close: () => ipcRenderer.send("window:close"),
  restart: () => ipcRenderer.send("app:restart"),
  
  // === System Information ===
  getPlatform: () => ipcRenderer.invoke("system:getPlatform"),
  getVersion: () => ipcRenderer.invoke("app:getVersion"),
  getPath: (name) => ipcRenderer.invoke("app:getPath", name),
  
  // === Communication ===
  send: (channel, data) => {
    // Whitelist channels for security
    const validChannels = ["toMain", "app:message", "data:sync"];
    if (validChannels.includes(channel)) {
      ipcRenderer.send(channel, data);
    }
  },
  
  receive: (channel, callback) => {
    // Whitelist channels for security
    const validChannels = ["fromMain", "app:update", "data:change"];
    if (validChannels.includes(channel)) {
      ipcRenderer.on(channel, (event, ...args) => callback(...args));
    }
  },
  
  // === Clipboard Operations ===
  copyToClipboard: (text) => ipcRenderer.invoke("clipboard:write", text),
  pasteFromClipboard: () => ipcRenderer.invoke("clipboard:read"),
  
  // === External Links ===
  openExternal: (url) => ipcRenderer.invoke("shell:openExternal", url),
  
  // === Storage (Example for app preferences) ===
  getPreference: (key) => ipcRenderer.invoke("storage:get", key),
  setPreference: (key, value) => ipcRenderer.invoke("storage:set", key, value),
  
  // === Native Notifications ===
  showNotification: (title, body) => ipcRenderer.send("notification:show", { title, body }),
  
  // === Auto-updater (if implemented) ===
  checkForUpdates: () => ipcRenderer.invoke("updater:check"),
  downloadUpdate: () => ipcRenderer.send("updater:download"),
  installUpdate: () => ipcRenderer.send("updater:install"),
});

// ===== OPTIONAL: EXPOSE APP INFO TO RENDERER =====
window.addEventListener("DOMContentLoaded", () => {
  console.log("[Electron Preload] DOM content loaded.");
  
  // AI AGENT NOTE: You can inject app information into the DOM here
  // Example: Display app version in footer
  /*
  ipcRenderer.invoke("app:getVersion").then(version => {
    const versionElement = document.getElementById("app-version");
    if (versionElement) {
      versionElement.textContent = `v${version}`;
    }
  });
  */
  
  // Example: Add platform-specific CSS class
  /*
  ipcRenderer.invoke("system:getPlatform").then(platform => {
    document.body.classList.add(`platform-${platform}`);
  });
  */
});

// ===== SECURITY NOTE =====
// NEVER expose sensitive Node.js APIs directly (like require, process, etc.)
// Always use IPC to communicate with the main process for privileged operations 
