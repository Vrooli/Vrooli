/**
 * The preload script runs before. It has access to web APIs
 * as well as Electron's renderer process modules and some
 * Node.js modules. 
 *
 * Learn more about the Electron preload script and context isolation:
 * https://www.electronjs.org/docs/latest/tutorial/process-model#preload-scripts
 */
const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('electronAPI', {
  // Expose a function to trigger the file open dialog and read the file
  openFile: () => ipcRenderer.invoke('dialog:openFile')
  // Add other APIs you want to expose here
});

window.addEventListener('DOMContentLoaded', () => {
  console.log('[Electron Preload] DOM content loaded.');

  // Example: Expose a safe subset of Node.js/Electron APIs to the renderer
  // const { contextBridge, ipcRenderer } = require('electron');
  // contextBridge.exposeInMainWorld('electronAPI', {
  //   sendMessage: (channel, data) => ipcRenderer.send(channel, data),
  //   // Add other APIs you want to expose here
  // });
}); 