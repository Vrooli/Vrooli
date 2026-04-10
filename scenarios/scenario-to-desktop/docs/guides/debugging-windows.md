# Debugging Windows Desktop Applications

This guide helps you debug Electron desktop applications built for Windows, especially when testing on a Windows machine after building on Linux.

## Common Issues

### 1. Application Doesn't Start (Silent Failure)

**Symptoms:**
- Double-clicking the .exe does nothing
- No error message appears
- Process starts briefly then immediately exits

**Most Common Causes:**
1. **Missing external servers** - App is configured to connect to servers that don't exist on Windows
2. **Port conflicts** - Required ports are already in use
3. **Missing dependencies** - Visual C++ Redistributable or other system libraries
4. **File path issues** - Hardcoded Linux paths that don't exist on Windows

### 2. How to Debug from Command Line

#### Method 1: Run the Unpacked App with Console Output

The easiest way to see what's happening is to run the unpacked executable directly from PowerShell or Command Prompt:

**PowerShell:**
```powershell
# Navigate to the unpacked app directory
cd "C:\path\to\app\win-unpacked"

# Run the exe - this will show console output
.\YourApp.exe

# Or redirect output to a file
.\YourApp.exe > debug.log 2>&1
```

**Command Prompt:**
```cmd
cd "C:\path\to\app\win-unpacked"
YourApp.exe
```

**Expected Output:**
- You should see console.log messages from the Electron main process
- Error messages will appear in the console
- The app window should appear if startup succeeds

#### Method 2: Enable DevTools in Production Build

Edit `src/main.ts` before building and change:

```typescript
// Change this:
ENABLE_DEV_TOOLS: true,

// Or in the main window creation:
if (APP_CONFIG.ENABLE_DEV_TOOLS && !app.isPackaged) {
    mainWindow.webContents.openDevTools();
}

// To this (always open DevTools):
if (APP_CONFIG.ENABLE_DEV_TOOLS) {
    mainWindow.webContents.openDevTools();
}
```

Then rebuild with `npm run dist:win`.

#### Method 3: Add More Logging

Add extensive logging to `src/main.ts`:

```typescript
// At the very start of the file, after imports:
console.log('[Desktop App] Starting application...');
console.log('[Desktop App] Platform:', process.platform);
console.log('[Desktop App] App path:', app.getAppPath());
console.log('[Desktop App] User data:', app.getPath('userData'));
console.log('[Desktop App] Config:', JSON.stringify(APP_CONFIG, null, 2));

// In app.whenReady():
app.whenReady().then(async () => {
    console.log('[Desktop App] App is ready');

    try {
        if (APP_CONFIG.ENABLE_SPLASH) {
            console.log('[Desktop App] Creating splash window...');
            createSplashWindow();
        }

        console.log('[Desktop App] Creating main window...');
        await createMainWindow();

        console.log('[Desktop App] Checking server...');
        // ... rest of your code
    } catch (error) {
        console.error('[Desktop App] Fatal error during startup:', error);
        dialog.showErrorBox('Startup Error', error.message || String(error));
        app.quit();
    }
});
```

### 3. Check Event Viewer for Crash Details

If the app crashes immediately, Windows Event Viewer may have details:

1. Press `Win + X` and select "Event Viewer"
2. Navigate to "Windows Logs" → "Application"
3. Look for errors from your app name
4. Check the error details for crash information

### 4. Server Configuration Issues

If your app expects to connect to external servers (like the scenario's UI/API servers), you have several options:

#### Option A: Run as Portable App (Not Standalone)

1. Start the scenario servers on Linux: `vrooli scenario start <scenario-name>`
2. Find the ports: `vrooli scenario status <scenario-name>`
3. Set up an SSH tunnel from Windows to Linux:
   ```powershell
   ssh -L 35000:localhost:35000 -L 15000:localhost:15000 user@linux-server
   ```
4. Run the desktop app on Windows - it will connect through the tunnel

#### Option B: Bundle Servers in Desktop App

Modify the desktop app configuration before generation to bundle the server:

```json
{
  "SERVER_TYPE": "static",  // or "node" or "executable"
  "SERVER_PATH": "path/to/bundled/ui/dist"
}
```

This requires regenerating the desktop app with the servers bundled.

#### Option C: Make the App Offline-First

Modify the app to work without server connections, storing data locally.

### 5. Missing Icon Issue

**Symptoms:**
- The .exe file has no icon (shows generic Windows icon)
- Installer might build correctly but exe has no icon

**Solution:**
This is usually caused by disabling `signAndEditExecutable` to work around rcedit/Wine issues. The fix involves:

1. Using the `afterPack` hook to manually embed the icon
2. Ensuring the `scripts/post-build-windows.js` script runs correctly
3. Checking that Wine and rcedit are properly installed

See the scenario-to-desktop template for the complete solution.

### 6. Build a Debug Version

Use the debug build script which creates an unpacked version without the installer:

```bash
# On Linux (where you're building)
npm run dist:win:debug
```

This creates a `dist-electron/win-unpacked/` directory. Copy this entire folder to Windows and run the .exe from PowerShell to see console output.

### 7. Network/Firewall Issues

**Symptoms:**
- App loads but shows "connection refused" or times out
- Can't connect to localhost servers

**Solutions:**

1. **Check Windows Firewall:**
   ```powershell
   # Allow app through firewall
   netsh advfirewall firewall add rule name="YourApp" dir=in action=allow program="C:\path\to\YourApp.exe" enable=yes
   ```

2. **Verify localhost resolution:**
   ```powershell
   ping localhost
   # Should resolve to 127.0.0.1
   ```

3. **Check if ports are in use:**
   ```powershell
   netstat -ano | findstr :35000
   ```

## Testing Checklist

When debugging a Windows desktop app:

- [ ] Run the .exe from PowerShell/CMD to see console output
- [ ] Check Windows Event Viewer for crash logs
- [ ] Verify external servers are accessible (if needed)
- [ ] Check that required ports aren't already in use
- [ ] Install Visual C++ Redistributable if needed
- [ ] Try running as Administrator (to rule out permissions)
- [ ] Check antivirus isn't blocking the app
- [ ] Verify the icon is embedded correctly
- [ ] Test with DevTools enabled
- [ ] Check app.log file in user data directory (if app creates one)

## Getting Help

When reporting issues, include:

1. Console output from running the .exe in PowerShell
2. Windows Event Viewer error details (if app crashes)
3. The app configuration (`APP_CONFIG` from `main.ts`)
4. Windows version (run `winver` command)
5. Whether the app works in dev mode (`npm run dev` on Linux)

## Scenario-Specific Notes

### Picker Wheel

- **Expected Behavior**: Connects to external UI server on port 35000 and API on port 15000
- **Windows Issue**: These servers don't exist on Windows, so the app waits 30 seconds then times out
- **Solution**: Use SSH tunnel (Option A above) or regenerate with bundled UI (Option B)

### System Monitor

- **Expected Behavior**: Monitors local system resources
- **Windows Issue**: May need elevated permissions for some metrics
- **Solution**: Run as Administrator or adjust which metrics are collected

---

Review this guide after major Electron/template changes or Windows packaging updates.
