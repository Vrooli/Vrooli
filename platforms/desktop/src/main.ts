import { app, BrowserWindow, net, shell, Menu } from "electron";
import { type ChildProcess, fork, spawn } from "node:child_process";
import path from "node:path";
import process from "node:process";

// ===== TEMPLATE CONFIGURATION - AI AGENTS SHOULD MODIFY THIS SECTION =====
// This template can work with different app architectures:
// 1. Node.js server (use fork)
// 2. Static files (set SERVER_TYPE to 'static')
// 3. External server (set SERVER_TYPE to 'external')
// 4. Bundled executable (set SERVER_TYPE to 'executable')

const APP_CONFIG = {
    // Application identity
    APP_NAME: "{{APP_NAME}}",  // Replace with your app name
    APP_URL: "{{APP_URL}}",     // Replace with your app's help/about URL
    
    // Server configuration
    SERVER_TYPE: "node",        // Options: 'node', 'static', 'external', 'executable'
    SERVER_PORT: 3000,           // Port your backend runs on (if applicable)
    SERVER_PATH: "{{SERVER_PATH}}", // Path to server entry point (relative to app root)
    // Examples:
    // - Node.js: "backend/dist/server.js"
    // - Static: "dist/index.html"
    // - External: "https://api.example.com"
    // - Executable: "bin/server.exe"
    
    // Window configuration
    WINDOW_WIDTH: 1200,
    WINDOW_HEIGHT: 800,
    WINDOW_BACKGROUND: "#2e2c29", // Default background color
    
    // Timing configuration
    SERVER_CHECK_INTERVAL_MS: 500,  // How often to check if server is ready
    SERVER_CHECK_TIMEOUT_MS: 30000, // Maximum time to wait for server
    
    // Features
    ENABLE_SPLASH: true,         // Show splash screen during startup
    ENABLE_MENU: true,           // Show application menu
    ENABLE_SINGLE_INSTANCE: true, // Prevent multiple instances
};

let serverProcess: ChildProcess | null = null;
let mainWindow: BrowserWindow | null = null;
let splashWindow: BrowserWindow | null = null;

const SERVER_URL = APP_CONFIG.SERVER_TYPE === 'external' 
    ? APP_CONFIG.SERVER_PATH 
    : `http://localhost:${APP_CONFIG.SERVER_PORT}`;

// --- Function Declarations (Moved to Top Level) ---

// Function to poll the server until it's ready
function checkServerReady(url: string, timeout: number): Promise<void> {
    return new Promise((resolve, reject) => {
        let elapsedTime = 0;
        const interval = setInterval(() => {
            const request = net.request(url);
            request.on("response", (response) => {
                // Any response means the server is up
                console.log(`[Electron Main] Server check successful (Status: ${response.statusCode}) at ${url}`);
                clearInterval(interval);
                request.abort(); // Abort the request as we just needed to know it's listening
                resolve();
            });
            request.on("error", (error) => {
                // Ignore connection refused, keep trying
                if (error.message.includes("ECONNREFUSED")) {
                    // console.log(`[Electron Main] Server check failed (ECONNREFUSED), retrying...`);
                } else {
                    console.error(`[Electron Main] Server check error: ${error.message}`);
                }
                request.abort();
            });
            request.end();

            elapsedTime += SERVER_CHECK_INTERVAL_MS;
            if (elapsedTime >= timeout) {
                clearInterval(interval);
                console.error(`[Electron Main] Server did not become ready at ${url} within ${timeout}ms.`);
                reject(new Error(`Server not ready within timeout: ${url}`));
            }
        }, SERVER_CHECK_INTERVAL_MS);
    });
}

// Function to create the main application window (hidden initially)
async function createWindow() {
    console.log("[Electron Main] Creating main window (hidden)...");
    mainWindow = new BrowserWindow({
        width: 1200,
        height: 800,
        show: false, // Keep hidden initially
        backgroundColor: "#2e2c29", // Match splash/app background
        webPreferences: {
            // AI AGENT NOTE: Ensure preload script is built to this location
            preload: path.join(app.getAppPath(), "desktop/dist/preload.js"),
            nodeIntegration: false,
            contextIsolation: true,
        },
    });

    mainWindow.on("closed", () => {
        console.log("[Electron Main] Main window closed.");
        mainWindow = null;
        // Consider if app should quit when main window is closed, even if server is running
        // if (process.platform !== 'darwin') app.quit();
    });

    // Load the main content - don't show yet
    try {
        console.log(`[Electron Main] Waiting for server to become ready at ${SERVER_URL} for main window...`);
        await checkServerReady(SERVER_URL, SERVER_CHECK_TIMEOUT_MS);
        console.log(`[Electron Main] Server ready. Loading main window URL: ${SERVER_URL}`);

        if (!mainWindow || mainWindow.isDestroyed()) {
            console.log("[Electron Main] Main window was closed before URL could be loaded.");
            return;
        }
        await mainWindow.loadURL(SERVER_URL);
        console.log("[Electron Main] Main window URL loaded successfully.");

        // Now that content is loaded, close splash and show main window
        if (splashWindow && !splashWindow.isDestroyed()) {
            console.log("[Electron Main] Closing splash window.");
            splashWindow.close();
            splashWindow = null;
        }
        if (mainWindow && !mainWindow.isDestroyed()) {
            console.log("[Electron Main] Showing main window.");
            mainWindow.show();
        } else {
            console.log("[Electron Main] Main window was destroyed before it could be shown.");
        }

    } catch (err) {
        console.error("[Electron Main] Error during main window load sequence:", (err as Error).message);
        // Close splash and show error in main window (if possible)
        if (splashWindow && !splashWindow.isDestroyed()) {
            splashWindow.close();
            splashWindow = null;
        }
        if (mainWindow && !mainWindow.isDestroyed()) {
            mainWindow.loadURL(`data:text/html;charset=utf-8,<h1>Application Error</h1><p>Failed to load application from ${SERVER_URL}. Error: ${(err as Error).message}</p>`);
            mainWindow.show(); // Show the main window with the error
        } else {
            // If main window is gone, maybe show a native error dialog and quit
            console.error("[Electron Main] Cannot show error in main window, quitting.");
            app.quit();
        }
    }
}

// Function to create the splash screen window
function createSplashWindow() {
    console.log("[Electron Main] Creating splash window...");
    splashWindow = new BrowserWindow({
        width: 400,
        height: 300,
        frame: false, // No window frame
        resizable: false,
        movable: false,
        center: true,
        backgroundColor: "#2e2c29",
        webPreferences: {
            // No node integration or preload needed for simple splash
        },
    });

    // AI AGENT NOTE: Update this path if your app structure differs
    const splashPath = path.join(app.getAppPath(), "desktop/src/splash.html");
    console.log(`[Electron Main] Loading splash screen from: ${splashPath}`);
    splashWindow.loadFile(splashPath)
        .then(() => console.log("[Electron Main] Splash screen loaded."))
        .catch(err => {
            console.error("[Electron Main] Failed to load splash screen:", err);
            // Handle splash screen load failure (e.g., close it and proceed, or quit)
            if (splashWindow && !splashWindow.isDestroyed()) splashWindow.close();
            splashWindow = null;
            // Maybe try creating main window directly with error or quit
            // createWindow().catch(e => { console.error("Failed to create main window after splash fail:", e); app.quit(); });
            app.quit(); // Simplest handling: quit if splash fails
        });

    splashWindow.on("closed", () => {
        splashWindow = null; // Dereference
    });
}

// Function to start the backend server
function startServer() {
    if (serverProcess) {
        console.log("[Electron Main] Server process already requested to start.");
        return; // Avoid starting multiple server processes
    }
    // AI AGENT NOTE: Adjust this section based on SERVER_TYPE
    const serverPath = path.resolve(app.getAppPath(), APP_CONFIG.SERVER_PATH);
    console.log(`[Electron Main] Attempting to start server from: ${serverPath}`);

    try {
        // Different server startup methods based on configuration
        if (APP_CONFIG.SERVER_TYPE === 'node') {
            // Fork a Node.js process
            serverProcess = fork(serverPath, [], {
            // Pass necessary environment variables, inherit stdio, etc.
            stdio: "inherit", // Show server logs in Electron's console
            // detached: true // Consider if detaching is needed, usually not for this pattern
        });

        serverProcess.on("error", (err) => {
            console.error("[Electron Main] Server process error:", err);
            // Handle error appropriately (e.g., show error message in UI, exit app)
            // Consider showing an error in the mainWindow if it exists
            if (mainWindow && !mainWindow.isDestroyed()) {
                mainWindow.loadURL(`data:text/html;charset=utf-8,<h1>Server Error</h1><p>The backend server process encountered an error: ${err.message}. The application might not function correctly.</p>`);
            }
            // Optionally quit if the server is critical and cannot start
            // app.quit();
            serverProcess = null; // Clear the reference on error
        });

        serverProcess.on("exit", (code, signal) => {
            console.log(`[Electron Main] Server process exited with code: ${code}, signal: ${signal}`);
            serverProcess = null; // Clear the reference on exit
            // Optionally attempt to restart or handle unexpected exit,
            // but be careful not to create loops. Maybe show an error in the UI.
            if (mainWindow && !mainWindow.isDestroyed()) {
                mainWindow.loadURL(`data:text/html;charset=utf-8,<h1>Server Stopped</h1><p>The backend server process stopped unexpectedly (code: ${code}, signal: ${signal}). The application might not function correctly.</p>`);
            }
            // Decide if the app should quit if the server dies
            // if (code !== 0 && signal !== 'SIGTERM') { // Example: Quit if not a clean exit
            //   app.quit();
            // }
        });
        } else if (APP_CONFIG.SERVER_TYPE === 'executable') {
            // Spawn an executable
            serverProcess = spawn(serverPath, [], {
                stdio: "inherit",
            });
        } else if (APP_CONFIG.SERVER_TYPE === 'static') {
            // No server process needed for static files
            console.log("[Electron Main] Static mode - no server process needed.");
            return;
        } else if (APP_CONFIG.SERVER_TYPE === 'external') {
            // External server - no local process to start
            console.log("[Electron Main] External server mode - no local process needed.");
            return;
        }

        console.log("[Electron Main] Server process requested to start successfully.");

    } catch (error) {
        console.error("[Electron Main] Critical error forking server process:", error);
        // Show error and quit if server cannot be forked at all
        if (mainWindow && !mainWindow.isDestroyed()) {
            mainWindow.loadURL(`data:text/html;charset=utf-8,<h1>Fatal Error</h1><p>Could not start the backend server process. Error: ${(error as Error).message}</p>`);
        } else {
            // If window isn't even created yet, just quit.
            app.quit();
        }
        serverProcess = null;
    }
}

// --- Menu Template ---
const menuTemplate: (Electron.MenuItemConstructorOptions | Electron.MenuItem)[] = [
    // { role: 'appMenu' } // On macOS
    ...(process.platform === "darwin" ? [{ role: "appMenu" as const }] : []),
    {
        role: "fileMenu",
        // Example customization:
        // label: 'File',
        // submenu: [
        //     process.platform === 'darwin' ? { role: 'close' } : { role: 'quit' }
        // ]
    },
    { role: "editMenu" },
    { role: "viewMenu" },
    { role: "windowMenu" },
    {
        role: "help",
        submenu: [
            {
                label: "Learn More",
                click: async () => {
                    await shell.openExternal(APP_CONFIG.APP_URL);
                },
            },
            // Add other help items like 'About' here
        ],
    },
];

// --- Single Instance Lock ---
const gotTheLock = app.requestSingleInstanceLock();

if (!gotTheLock) {
    console.log("[Electron Main] Another instance is already running. Quitting.");
    app.quit();
} else {
    app.on("second-instance", (event, commandLine, workingDirectory) => {
        console.log("[Electron Main] Second instance detected.");
        // Focus main window if it exists and is ready, otherwise maybe focus splash?
        if (mainWindow && !mainWindow.isDestroyed() && mainWindow.isVisible()) {
            console.log("[Electron Main] Focusing existing main window.");
            if (mainWindow.isMinimized()) mainWindow.restore();
            mainWindow.focus();
        } else if (splashWindow && !splashWindow.isDestroyed()) {
            console.log("[Electron Main] Focusing existing splash window.");
            splashWindow.focus();
        }
    });

    // Create mainWindow, load the rest of the app, etc...
    // The rest of the app lifecycle setup goes inside this 'else' block.

    // --- Electron App Lifecycle Events ---

    // This method will be called when Electron has finished
    // initialization and is ready to create browser windows.
    // Some APIs can only be used after this event occurs.
    app.whenReady().then(async () => {
        console.log("[Electron Main] App is ready.");

        // Build and set the application menu
        const menu = Menu.buildFromTemplate(menuTemplate);
        Menu.setApplicationMenu(menu);
        console.log("[Electron Main] Application menu set.");

        createSplashWindow(); // Show splash screen ASAP
        startServer(); // Start the backend server in parallel
        // Create the main window but don't wait for it to finish loading here
        // Its loading logic (including server check and showing) handles the swap.
        createWindow().catch(e => {
            console.error("Error during initial createWindow call:", e);
            if (splashWindow && !splashWindow.isDestroyed()) splashWindow.close();
            // Show native error dialog?
            app.quit();
        });

        app.on("activate", async () => {
            console.log("[Electron Main] Activate event triggered.");
            // On macOS it's common to re-create a window.
            // If main window exists, focus it. If not, start the create process.
            if (mainWindow && !mainWindow.isDestroyed()) {
                console.log("[Electron Main] Activate: Focusing main window.");
                if (mainWindow.isMinimized()) mainWindow.restore();
                mainWindow.focus();
            } else if (!splashWindow || splashWindow.isDestroyed()) {
                // Only create if no splash AND no main window exists/is loading
                console.log("[Electron Main] Activate: No windows open, creating splash & main.");
                createSplashWindow();
                createWindow().catch(e => { console.error("Error during activate createWindow call:", e); app.quit(); });
            } else {
                console.log("[Electron Main] Activate: Splash window still visible, focusing it.");
                splashWindow.focus();
            }
        });
    });

    // Quit when all windows are closed (might need adjustment depending on desired macOS behavior)
    app.on("window-all-closed", () => {
        console.log("[Electron Main] All windows closed (main and splash).");
        // This might now trigger earlier if only the splash window is closed due to an error.
        // Typically, you only quit if the *main* interaction window is closed.
        // Let's adjust: only quit if the main window reference is null AFTER it was potentially created.
        if (process.platform !== "darwin") {
            // Check if mainWindow was ever created or if it's just the splash that closed
            if (mainWindow === null) { // If mainWindow is null, it means it was closed normally or never opened fully
                console.log("[Electron Main] Main window closed or never fully opened. Quitting app (non-macOS).");
                app.quit();
            } else {
                console.log("[Electron Main] Assuming splash/error window closed, main window may still exist or be loading.");
            }
        } else {
            console.log("[Electron Main] Keeping app active (macOS).");
        }
    });

    // Ensure the server process is killed when the app quits.
    app.on("quit", () => {
        console.log("[Electron Main] App quit event triggered.");
        if (serverProcess && !serverProcess.killed) {
            console.log("[Electron Main] Attempting to kill server process...");
            const killed = serverProcess.kill(); // Standard kill signal (SIGTERM)
            console.log(`[Electron Main] Server process kill attempt returned: ${killed}`);
            serverProcess = null;
        } else {
            console.log("[Electron Main] Server process already null or killed.");
        }
        console.log("[Electron Main] Exiting application.");
    });

    // You can include the rest of your app's specific main process
    // code here. You can also put them in separate files and import them.
} // End of the 'else' block for single instance lock 
