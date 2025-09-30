const { app, BrowserWindow, globalShortcut, ipcMain, Tray, Menu, screen, shell } = require('electron');
const path = require('path');
const screenshot = require('screenshot-desktop');
const Store = require('electron-store');
const axios = require('axios');
const fs = require('fs').promises;
const os = require('os');

// Initialize persistent store
const store = new Store();

// Global variables
let mainWindow = null;
let tray = null;
let isCapturing = false;
const API_BASE_URL = process.env.API_PORT ? `http://localhost:${process.env.API_PORT}` : 'http://localhost:3250';

// Platform-specific hotkey
const HOTKEY = process.platform === 'darwin' ? 'Cmd+Shift+Space' : 'Ctrl+Shift+Space';

// Create the overlay window
function createOverlay() {
  const display = screen.getPrimaryDisplay();
  const { width, height } = display.bounds;

  mainWindow = new BrowserWindow({
    width: 600,
    height: 400,
    x: Math.floor((width - 600) / 2),
    y: Math.floor((height - 400) / 2),
    frame: false,
    transparent: true,
    alwaysOnTop: true,
    resizable: false,
    skipTaskbar: true,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      preload: path.join(__dirname, 'preload.js')
    },
    show: false
  });

  mainWindow.loadFile('overlay.html');

  // Hide instead of close
  mainWindow.on('close', (event) => {
    if (!app.isQuitting) {
      event.preventDefault();
      mainWindow.hide();
    }
    return false;
  });

  // Hide on blur (when clicking outside)
  mainWindow.on('blur', () => {
    if (!isCapturing) {
      mainWindow.hide();
    }
  });

  return mainWindow;
}

// Create system tray
function createTray() {
  // Simple icon as base64 (1x1 transparent PNG for now)
  const iconPath = path.join(__dirname, 'assets', 'tray-icon.png');
  
  tray = new Tray(iconPath);
  
  const contextMenu = Menu.buildFromTemplate([
    {
      label: 'Capture Issue',
      accelerator: HOTKEY,
      click: () => toggleOverlay()
    },
    {
      label: 'View History',
      click: () => openHistory()
    },
    { type: 'separator' },
    {
      label: 'Settings',
      click: () => openSettings()
    },
    {
      label: 'About',
      click: () => {
        shell.openExternal('https://github.com/Vrooli/scenarios/vrooli-assistant');
      }
    },
    { type: 'separator' },
    {
      label: 'Quit',
      click: () => {
        app.isQuitting = true;
        app.quit();
      }
    }
  ]);

  tray.setToolTip('Vrooli Assistant - Press ' + HOTKEY + ' to capture');
  tray.setContextMenu(contextMenu);
  
  // Click to show overlay
  tray.on('click', () => {
    toggleOverlay();
  });
}

// Toggle overlay visibility
function toggleOverlay() {
  if (mainWindow) {
    if (mainWindow.isVisible()) {
      mainWindow.hide();
    } else {
      // Position near cursor
      const point = screen.getCursorScreenPoint();
      const display = screen.getDisplayNearestPoint(point);
      const bounds = display.bounds;
      
      // Center on current display
      const windowWidth = 600;
      const windowHeight = 400;
      const x = Math.floor(bounds.x + (bounds.width - windowWidth) / 2);
      const y = Math.floor(bounds.y + (bounds.height - windowHeight) / 2);
      
      mainWindow.setPosition(x, y);
      mainWindow.show();
      mainWindow.focus();
    }
  }
}

// Capture screenshot
async function captureScreenshot() {
  isCapturing = true;
  mainWindow.hide();
  
  try {
    // Wait a bit for window to fully hide
    await new Promise(resolve => setTimeout(resolve, 100));
    
    // Capture all screens
    const screenshots = await screenshot.all();
    
    // Save screenshots to temp directory
    const tempDir = path.join(os.tmpdir(), 'vrooli-assistant');
    await fs.mkdir(tempDir, { recursive: true });
    
    const screenshotPaths = [];
    for (let i = 0; i < screenshots.length; i++) {
      const screenshotPath = path.join(tempDir, `screenshot-${Date.now()}-${i}.png`);
      await fs.writeFile(screenshotPath, screenshots[i]);
      screenshotPaths.push(screenshotPath);
    }
    
    // Show overlay again
    mainWindow.show();
    isCapturing = false;
    
    return screenshotPaths[0]; // Return primary screenshot
  } catch (error) {
    console.error('Screenshot failed:', error);
    mainWindow.show();
    isCapturing = false;
    throw error;
  }
}

// Get current scenario context
async function getCurrentContext() {
  try {
    // Get current URL from system if possible
    const activeWindow = BrowserWindow.getFocusedWindow();
    const url = activeWindow ? activeWindow.webContents.getURL() : 'unknown';
    
    // Try to detect scenario from URL
    const scenarioMatch = url.match(/localhost:(\d+)/);
    const scenarioName = scenarioMatch ? await detectScenarioByPort(scenarioMatch[1]) : 'unknown';
    
    return {
      url,
      scenario: scenarioName,
      timestamp: new Date().toISOString(),
      platform: process.platform,
      displays: screen.getAllDisplays().length
    };
  } catch (error) {
    console.error('Failed to get context:', error);
    return {
      url: 'unknown',
      scenario: 'unknown',
      timestamp: new Date().toISOString()
    };
  }
}

// Detect scenario by port (simplified)
async function detectScenarioByPort(port) {
  // This would normally query the API to map port to scenario
  // For now, return a placeholder
  return `scenario-on-port-${port}`;
}

// Submit issue to API
async function submitIssue(data) {
  try {
    const response = await axios.post(`${API_BASE_URL}/api/v1/assistant/capture`, data);
    return response.data;
  } catch (error) {
    console.error('Failed to submit issue:', error);
    throw error;
  }
}

// Spawn agent with context
async function spawnAgent(issueData, agentType = 'claude-code') {
  try {
    const response = await axios.post(`${API_BASE_URL}/api/v1/assistant/spawn-agent`, {
      issue_id: issueData.issue_id,
      agent_type: agentType,
      context: issueData.context,
      screenshot: issueData.screenshot_path,
      description: issueData.description
    });
    return response.data;
  } catch (error) {
    console.error('Failed to spawn agent:', error);
    throw error;
  }
}

// Open history view
function openHistory() {
  const historyWindow = new BrowserWindow({
    width: 800,
    height: 600,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      preload: path.join(__dirname, 'preload.js')
    }
  });
  
  historyWindow.loadFile('history.html');
}

// Open settings
function openSettings() {
  const settingsWindow = new BrowserWindow({
    width: 600,
    height: 400,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      preload: path.join(__dirname, 'preload.js')
    }
  });
  
  settingsWindow.loadFile('settings.html');
}

// IPC handlers
ipcMain.handle('capture-screenshot', async () => {
  return await captureScreenshot();
});

ipcMain.handle('get-context', async () => {
  return await getCurrentContext();
});

ipcMain.handle('submit-issue', async (event, data) => {
  return await submitIssue(data);
});

ipcMain.handle('spawn-agent', async (event, issueData, agentType) => {
  return await spawnAgent(issueData, agentType);
});

ipcMain.handle('hide-overlay', () => {
  mainWindow.hide();
});

ipcMain.handle('get-settings', () => {
  return {
    hotkey: store.get('hotkey', HOTKEY),
    apiUrl: store.get('apiUrl', API_BASE_URL),
    defaultAgentType: store.get('defaultAgentType', 'claude-code'),
    autoSpawnAgent: store.get('autoSpawnAgent', true)
  };
});

ipcMain.handle('save-settings', (event, settings) => {
  Object.keys(settings).forEach(key => {
    store.set(key, settings[key]);
  });
  
  // Re-register hotkey if changed
  if (settings.hotkey && settings.hotkey !== HOTKEY) {
    globalShortcut.unregister(HOTKEY);
    registerHotkey(settings.hotkey);
  }
  
  return true;
});

// Register global hotkey
function registerHotkey(hotkey = HOTKEY) {
  const success = globalShortcut.register(hotkey, () => {
    toggleOverlay();
  });
  
  if (!success) {
    console.error('Failed to register hotkey:', hotkey);
  } else {
    console.log('Hotkey registered:', hotkey);
  }
  
  return success;
}

// App event handlers
app.whenReady().then(() => {
  // Create overlay window
  createOverlay();
  
  // Create tray icon
  createTray();
  
  // Register global hotkey
  registerHotkey();
  
  // Handle test mode
  if (process.argv.includes('--test')) {
    console.log('Running in test mode');
    // Run tests and exit
    setTimeout(() => {
      app.quit();
    }, 2000);
  }

  // Handle test-hotkey mode
  if (process.argv.includes('--test-hotkey')) {
    console.log('Testing global hotkey registration...');
    // Hotkey should be registered at this point by registerHotkey()
    setTimeout(() => {
      const registered = globalShortcut.isRegistered(HOTKEY);
      if (registered) {
        console.log('Hotkey registered successfully');
        console.log(`Hotkey: ${HOTKEY}`);
      } else {
        console.error('Failed to register hotkey');
      }
      app.quit();
    }, 500);
  }

  // Handle daemon mode
  if (process.argv.includes('--daemon')) {
    console.log('Running in daemon mode');
    // Keep running in background
  }
});

app.on('window-all-closed', (event) => {
  // Prevent app from quitting when overlay is hidden
  event.preventDefault();
});

app.on('will-quit', () => {
  // Unregister all shortcuts
  globalShortcut.unregisterAll();
});

// Prevent multiple instances
const gotTheLock = app.requestSingleInstanceLock();

if (!gotTheLock) {
  app.quit();
} else {
  app.on('second-instance', () => {
    // Show overlay when second instance is attempted
    if (mainWindow) {
      toggleOverlay();
    }
  });
}

// Export for testing
module.exports = {
  createOverlay,
  toggleOverlay,
  captureScreenshot,
  getCurrentContext
};