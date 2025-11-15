# Desktop Feature Examples

This folder contains comprehensive, copy-paste-ready examples for adding native desktop features to your Electron apps.

## 📁 Available Examples

### 1. [file-operations.ts](./file-operations.ts)
**Add file system access to your app**

Features:
- ✅ Save files with native dialog
- ✅ Open files with native dialog
- ✅ Save to specific path (no dialog)
- ✅ Read from specific path
- ✅ Select directories
- ✅ Select multiple files
- ✅ React component examples
- ✅ Graceful web fallbacks

**Use when**: Your app needs to save/load files, export data, or manage documents.

### 2. [native-menus.ts](./native-menus.ts)
**Add native application menus**

Features:
- ✅ File menu (New, Open, Save, Export)
- ✅ Edit menu (Undo, Redo, Cut, Copy, Paste)
- ✅ View menu (Zoom, DevTools, Fullscreen)
- ✅ Keyboard shortcuts (Ctrl+S, Ctrl+N, etc.)
- ✅ Platform-specific menus (macOS app menu)
- ✅ Dynamic menu updates
- ✅ Right-click context menus

**Use when**: You want professional menu bars with keyboard shortcuts.

### 3. [notifications.ts](./notifications.ts)
**Show native OS notifications**

Features:
- ✅ Basic notifications
- ✅ Notifications with action buttons
- ✅ Notification click handling
- ✅ Action button click handling
- ✅ React hooks for notifications
- ✅ Notification manager (queue, priority)
- ✅ Web notification fallbacks

**Use when**: You need to alert users about events, task completion, or errors.

## 🚀 How to Use These Examples

### Quick Start

1. **Choose the feature** you want to add (file operations, menus, or notifications)

2. **Open the example file** and follow the step-by-step instructions

3. **Copy the code** to the appropriate files:
   - Preload code → `src/preload.ts`
   - Main process code → `src/main.ts`
   - React components → Your UI components

4. **Add TypeScript declarations** to get type safety

5. **Test** in both desktop and web modes (if applicable)

### Example Workflow: Adding File Save

```typescript
// 1. Add to preload.ts
contextBridge.exposeInMainWorld('electronAPI', {
  saveFile: (filename: string, content: string) =>
    ipcRenderer.invoke('save-file', filename, content),
});

// 2. Add to main.ts
ipcMain.handle('save-file', async (event, filename, content) => {
  const { filePath } = await dialog.showSaveDialog({
    defaultPath: filename
  });
  if (filePath) {
    await fs.writeFile(filePath, content);
    return { success: true, path: filePath };
  }
  return { success: false };
});

// 3. Use in React component
function SaveButton({ data }) {
  const handleSave = async () => {
    const result = await window.electronAPI.saveFile(
      'export.json',
      JSON.stringify(data)
    );
    if (result.success) {
      alert(`Saved to ${result.path}`);
    }
  };
  return <button onClick={handleSave}>Save</button>;
}
```

## 📖 Documentation Structure

Each example file contains:

1. **Overview** - What the example does
2. **Step-by-step instructions** - Exactly where to add each piece of code
3. **TypeScript declarations** - For type safety
4. **React components** - Real-world usage examples
5. **Advanced patterns** - Dynamic menus, notification queues, etc.
6. **Usage tips** - Best practices and gotchas

## 🎯 Common Patterns

### Graceful Degradation

All examples support both desktop and web modes:

```typescript
if (window.electronAPI) {
  // Desktop mode - use native features
  await window.electronAPI.saveFile('data.json', content);
} else {
  // Web mode - use browser fallback
  const blob = new Blob([content]);
  const url = URL.createObjectURL(blob);
  // ... download via <a> tag
}
```

### Error Handling

Always handle both success and error cases:

```typescript
const result = await window.electronAPI.saveFile(filename, content);

if (result.success) {
  console.log('Saved to:', result.path);
} else {
  console.error('Save failed:', result.error);
}
```

### TypeScript Support

Add declarations for type safety:

```typescript
declare global {
  interface Window {
    electronAPI?: {
      saveFile: (filename: string, content: string) => Promise<{
        success: boolean;
        path?: string;
        error?: string;
      }>;
    };
  }
}
```

## 🔐 Security Best Practices

### 1. Never Allow Arbitrary File System Access

```typescript
// ❌ Bad - allows reading any file
ipcMain.handle('read-file', async (event, filePath) => {
  return fs.readFile(filePath);
});

// ✅ Good - user chooses file with dialog
ipcMain.handle('open-file', async () => {
  const { filePaths } = await dialog.showOpenDialog({...});
  if (filePaths[0]) {
    return fs.readFile(filePaths[0]);
  }
});
```

### 2. Validate All Inputs

```typescript
ipcMain.handle('save-file', async (event, filename, content) => {
  // Validate filename
  if (!filename || filename.includes('..')) {
    return { success: false, error: 'Invalid filename' };
  }

  // Validate content
  if (typeof content !== 'string') {
    return { success: false, error: 'Invalid content type' };
  }

  // Proceed with save...
});
```

### 3. Use Context Isolation

Ensure your `BrowserWindow` has:

```typescript
const mainWindow = new BrowserWindow({
  webPreferences: {
    preload: path.join(__dirname, 'preload.js'),
    contextIsolation: true,  // ✅ Must be true
    nodeIntegration: false   // ✅ Must be false
  }
});
```

## 🎨 Combining Features

You can combine multiple features in one app:

```typescript
// preload.ts - Export all features
contextBridge.exposeInMainWorld('electronAPI', {
  // File operations
  saveFile: (...args) => ipcRenderer.invoke('save-file', ...args),
  openFile: () => ipcRenderer.invoke('open-file'),

  // Notifications
  showNotification: (...args) => ipcRenderer.invoke('show-notification', ...args),

  // Menu events
  onMenuSave: (callback) => ipcRenderer.on('menu-save', callback),

  // Add more as needed...
});
```

## 🧪 Testing Desktop Features

### Test on All Platforms

Desktop apps behave differently on Windows, macOS, and Linux:

- **Windows**: Test notifications, menus, file dialogs
- **macOS**: Test app menu, dock integration
- **Linux**: Test with different desktop environments

### Test Web Fallbacks

Make sure your app still works in the browser:

```bash
# Test in browser
cd ui && npm run dev

# Test in desktop
cd platforms/electron && npm run dev
```

### Test Edge Cases

- User cancels file dialog
- No write permission to directory
- File already exists
- Large files
- Special characters in filenames

## 📚 Additional Resources

- [Electron IPC Documentation](https://www.electronjs.org/docs/latest/tutorial/ipc)
- [Electron Security Best Practices](https://www.electronjs.org/docs/latest/tutorial/security)
- [Desktop Wrapper Guide](../DESKTOP_WRAPPER_GUIDE.md)
- [Electron Menu Documentation](https://www.electronjs.org/docs/latest/api/menu)
- [Electron Notification Documentation](https://www.electronjs.org/docs/latest/api/notification)

## 🤝 Contributing

Found a useful pattern not covered here? Add it! Examples should:

- Be well-documented with comments
- Include step-by-step instructions
- Show React component usage
- Handle errors gracefully
- Support both desktop and web modes
- Follow security best practices

## 💡 Tips for Success

1. **Start Simple**: Add one feature at a time
2. **Test Early**: Test on target platforms frequently
3. **Graceful Degradation**: Always provide web fallbacks
4. **Type Safety**: Use TypeScript declarations
5. **Security First**: Never trust renderer process input
6. **User Experience**: Use native dialogs and patterns
7. **Performance**: Minimize IPC calls, batch operations
8. **Accessibility**: Support keyboard shortcuts and screen readers

---

**Happy Desktop App Building!** 🚀

For questions or issues, see the main [scenario-to-desktop README](../../../../README.md).
