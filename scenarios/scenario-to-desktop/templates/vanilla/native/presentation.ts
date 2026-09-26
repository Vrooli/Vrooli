import { app, globalShortcut, ipcMain, screen, type BrowserWindow, type IpcMainInvokeEvent, type Rectangle } from 'electron';

import type { WindowState } from '../window-state/types';

export interface PresentationController {
    /** Compact windows restart expanded using the last expanded geometry. */
    persistenceState(): WindowState | null;
    reveal(mode: 'expanded' | 'palette' | 'pill'): void;
    setBackground(enabled: boolean): boolean;
    prepareQuit(): void;
    requestQuit(): boolean;
}

export interface ActivationImagePreview {
    contextId: string; expiresAt: number; mimeType: 'image/png'; dataUrl: string;
    sourceBounds: {x:number;y:number;width:number;height:number};
    /** Frozen capture provenance, never an observation lease or input authority. */
    source: {
        surface: {target:{ownerScenario:string;resourceId:string;hostNodeId:string};ownerScenario:string;surfaceId:string};
        captureId:string; displayId:string; geometryRevision:string; capturedAt:string;
        bounds:{x:number;y:number;width:number;height:number};
    };
}

export interface ActivationReference {
    contextId: string; expiresAt: number;
    /** Trusted closure bound to the capture session; never projected into IPC. */
    discard?: () => Promise<void>;
    readImage?: (signal: AbortSignal) => Promise<ActivationImagePreview>;
}

export interface ActivationCapture {
    /** Trusted main-process owner adapter; never supplied through renderer IPC. */
    capture(signal: AbortSignal): Promise<ActivationReference>;
}

type Mode = 'expanded' | 'palette' | 'pill' | 'hidden';
interface Extension { version: number; module: string; permissions: string[]; activation_shortcut?: string }

/** One window and one renderer: no reload, navigation or duplicate application state. */
export function installPresentation(window: BrowserWindow, extension: Extension | null, activationCapture?: ActivationCapture): PresentationController | undefined {
    if (!extension) return;
    const permissions = extension.version === 3 ? ['window.presentation', 'global-shortcut', 'desktop.context'] : extension.version === 2 ? ['window.presentation', 'global-shortcut'] : ['window.presentation'];
    if ((extension.version !== 1 && extension.version !== 2 && extension.version !== 3) || extension.module !== 'presentation' ||
        extension.permissions.length !== permissions.length || permissions.some(p => !extension.permissions.includes(p)) ||
        (extension.version === 1 && extension.activation_shortcut !== undefined) ||
        (extension.version >= 2 && (!extension.activation_shortcut || !/^(CommandOrControl|Control|Alt|Super)(\+(Shift|Alt))?\+(Space|[A-Z0-9])$/.test(extension.activation_shortcut)))) {
        throw new Error('unsupported native presentation extension');
    }
    if (activationCapture && extension.version !== 3) throw new Error('activation capture requires desktop.context permission');
    let trustedDocument: string | null = null;
    const documentIdentity = () => {
        const url = new URL(window.webContents.getURL());
        return url.protocol === 'file:' ? `${url.origin}${url.pathname}` : url.origin;
    };
    window.webContents.on('did-finish-load', () => {
        if (trustedDocument === null) trustedDocument = documentIdentity();
        if (documentIdentity() === trustedDocument) window.webContents.send('native:presentation:ready');
    });
    let activationAbort: AbortController | undefined;
    let activationContext: ActivationReference | undefined;
    let activationStatus: 'unavailable' | 'capturing' | 'ready' = 'unavailable';
    const cancelActivation = () => { activationAbort?.abort(); activationAbort = undefined; if (activationStatus === 'capturing') activationStatus = 'unavailable'; };
    let mode: Mode = 'expanded';
    let revision = 0;
    let background = false;
    let quitting = false;
    let quitGuard = false;
    let quitSequence = 0;
    let pendingQuit = 0;
    let shortcutStatus: 'disabled' | 'registered' | 'unavailable' = 'disabled';
    let accelerator = extension.activation_shortcut ?? '';
    const snapshot = () => ({ version: 1, mode, revision, canHide: shortcutStatus === 'registered', shortcut: { accelerator, status: shortcutStatus }, ...(activationCapture ? { activation: { status: activationContext && activationContext.expiresAt <= Date.now() ? 'unavailable' : activationStatus, ...(activationContext && activationContext.expiresAt > Date.now() ? { contextId: activationContext.contextId, expiresAt: activationContext.expiresAt, ...(activationContext.readImage ? {hasImage:true} : {}) } : {}) } } : {}) });
    const publish = () => {
        if (!window.isDestroyed() && trustedDocument !== null && documentIdentity() === trustedDocument) window.webContents.send('native:presentation:changed', snapshot());
    };
    let expanded = window.getBounds();
    let maximized = false;
    let fullscreen = false;
    const initialMinimum = window.getMinimumSize();
    const initialResizable = window.isResizable();
    const initialAlwaysOnTop = window.isAlwaysOnTop();
    const initialMenuVisible = window.isMenuBarVisible();
    const authorized = (event: IpcMainInvokeEvent) => {
        if (window.isDestroyed() || event.sender !== window.webContents || event.senderFrame !== window.webContents.mainFrame ||
            trustedDocument === null || documentIdentity() !== trustedDocument) {
            throw new Error('presentation sender is not the application main frame');
        }
    };
    const clamp = (bounds: Rectangle): Rectangle => {
        const area = screen.getDisplayMatching(bounds).workArea;
        const width = Math.min(bounds.width, area.width);
        const height = Math.min(bounds.height, area.height);
        return { width, height, x: Math.max(area.x, Math.min(bounds.x, area.x + area.width - width)),
            y: Math.max(area.y, Math.min(bounds.y, area.y + area.height - height)) };
    };
    const visible = () => {
        if (!window.isDestroyed() && !window.isFullScreen() && !window.isMaximized()) window.setBounds(clamp(window.getBounds()));
    };
    const applyMode = (requested: unknown) => {
        if (requested !== 'expanded' && requested !== 'palette' && requested !== 'pill' && requested !== 'hidden') throw new Error('unsupported presentation mode');
        if (requested === 'hidden' && shortcutStatus !== 'registered') throw new Error('hidden presentation requires a registered activation shortcut');
        if (window.isFullScreen() && requested !== 'hidden') throw new Error('leave fullscreen before switching presentation');
        if (requested === mode) return snapshot();
        if (mode === 'expanded') {
            expanded = window.getNormalBounds();
            maximized = window.isMaximized();
            fullscreen = window.isFullScreen();
            if (maximized && requested !== 'hidden') window.unmaximize();
        }
        if (requested === 'hidden') {
            window.hide();
            mode = requested;
            revision++;
            publish();
            return snapshot();
        }
        const wasHidden = mode === 'hidden';
        const compact = requested !== 'expanded';
        if (wasHidden && compact && window.isMaximized()) window.unmaximize();
        window.setMinimumSize(compact ? 160 : (initialMinimum[0] ?? 0), compact ? 64 : (initialMinimum[1] ?? 0));
        window.setResizable(compact ? false : initialResizable);
        window.setAlwaysOnTop(compact || initialAlwaysOnTop);
        window.setMenuBarVisibility(compact ? false : initialMenuVisible);
        const bounds = compact ? { ...window.getBounds(), width: requested === 'pill' ? 320 : 560, height: requested === 'pill' ? 96 : 480 } : expanded;
        window.setBounds(clamp(bounds));
        if (!compact && maximized) window.maximize();
        mode = requested;
        if (wasHidden) window.show();
        revision++;
        publish();
        return snapshot();
    };
    ipcMain.handle('native:presentation:get', event => { authorized(event); return snapshot(); });
    let previewRead: Promise<ActivationImagePreview> | undefined;
    ipcMain.handle('native:presentation:read-context-image', async event => {
        authorized(event);
        const current = activationContext;
        if (!current?.readImage || current.expiresAt <= Date.now()) throw new Error('context image unavailable');
        if (previewRead) throw new Error('context image read already pending');
        const reading = current.readImage(AbortSignal.timeout(2000));
        previewRead = reading;
        try {
            const result = await reading;
            authorized(event);
            if (activationContext !== current || current.expiresAt <= Date.now() || result.contextId !== current.contextId || result.expiresAt !== current.expiresAt) throw new Error('context image expired or dismissed');
            return result;
        } finally { if (previewRead === reading) previewRead = undefined; }
    });
    ipcMain.handle('native:presentation:dismiss-context', async event => {
        authorized(event);
        if (!activationCapture) throw new Error('activation context is unavailable');
        cancelActivation();
        const dismissed = activationContext;
        activationContext = undefined; activationStatus = 'unavailable'; revision++;
        publish();
        await dismissed?.discard?.();
        return snapshot();
    });
    ipcMain.handle('native:presentation:set', (event, requested: unknown) => { authorized(event); cancelActivation(); return applyMode(requested); });
    const reveal = (requested: 'expanded' | 'palette' | 'pill') => {
        cancelActivation();
        if (window.isDestroyed() || trustedDocument === null || documentIdentity() !== trustedDocument) return;
        if (window.isMinimized()) window.restore();
        if (!window.isFullScreen()) applyMode(requested);
        else if (mode === 'hidden') { mode = 'expanded'; revision++; }
        window.show();
        window.focus();
        publish();
    };
    const requestQuit = () => {
        if (quitting || !quitGuard) return true;
        if (!pendingQuit) {
            pendingQuit = ++quitSequence;
        }
        reveal('expanded');
        window.webContents.send('native:presentation:quit-request', pendingQuit);
        return false;
    };
    ipcMain.handle('native:presentation:quit-guard', event => {
        authorized(event); quitGuard = true; return pendingQuit;
    });
    ipcMain.handle('native:presentation:quit-decision', (event, id: unknown, decision: unknown) => {
        authorized(event);
        if (!pendingQuit || id !== pendingQuit) throw new Error('stale quit decision');
        if (decision !== 'quit' && decision !== 'cancel' && decision !== 'background') throw new Error('unsupported quit decision');
        if (decision === 'background') applyMode('hidden');
        pendingQuit = 0;
        if (decision === 'quit') { quitting = true; queueMicrotask(() => app.quit()); }
        return snapshot();
    });
    const activate = () => {
        if (!activationCapture) { reveal('palette'); return; }
        if (quitting || window.isDestroyed() || trustedDocument === null || documentIdentity() !== trustedDocument) return;
        if (activationAbort) return; // Repeated shortcut presses share one capture.
        const abort = new AbortController(); activationAbort = abort;
        activationContext = undefined; activationStatus = 'capturing'; revision++; publish();
        void (async () => {
            let timer: ReturnType<typeof setTimeout> | undefined;
            try {
                const result = await Promise.race([
                    activationCapture.capture(abort.signal).then(result => {
                        if (abort.signal.aborted) {
                            void result.discard?.().catch(() => { /* Helper lifecycle bounds retention if deletion fails. */ });
                            throw new Error('activation capture cancelled');
                        }
                        return result;
                    }),
                    new Promise<never>((_, reject) => { timer = setTimeout(() => { reject(new Error('activation capture timeout')); }, 750); }),
                ]);
                if (abort.signal.aborted) return;
                if (!/^[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}$/.test(result.contextId) || result.contextId === '00000000-0000-0000-0000-000000000000' || !Number.isSafeInteger(result.expiresAt) || result.expiresAt <= Date.now() || result.expiresAt > Date.now() + 30000) throw new Error('invalid activation reference');
                activationContext = { contextId: result.contextId, expiresAt: result.expiresAt, ...(result.discard ? { discard: result.discard } : {}), ...(result.readImage ? {readImage:result.readImage} : {}) };
                activationStatus = 'ready';
            } catch { if (activationAbort === abort) activationStatus = 'unavailable'; }
            finally {
                if (timer !== undefined) clearTimeout(timer);
                if (activationAbort === abort) {
                    cancelActivation();
                    revision++;
                    reveal('palette');
                }
            }
        })();
    };
    window.webContents.on('did-start-navigation', cancelActivation);
    if (extension.version >= 2 && extension.activation_shortcut) {
        try {
            const registered = globalShortcut.register(extension.activation_shortcut, activate);
            shortcutStatus = registered ? 'registered' : 'unavailable';
        } catch { shortcutStatus = 'unavailable'; }
    }
    ipcMain.handle('native:presentation:shortcut', (event, requested: unknown) => {
        authorized(event);
        if (extension.version < 2 || typeof requested !== 'string' ||
            !/^(CommandOrControl|Control|Alt|Super)(\+(Shift|Alt))?\+(Space|[A-Z0-9])$/.test(requested)) {
            throw new Error('unsupported activation shortcut');
        }
        if (shortcutStatus === 'registered' && requested === accelerator) return snapshot();
        // Keep the working recovery path until its replacement is registered.
        let registered = false;
        try { registered = globalShortcut.register(requested, activate); } catch { /* Desktop refused. */ }
        if (!registered) throw new Error('activation shortcut is unavailable');
        if (shortcutStatus === 'registered') globalShortcut.unregister(accelerator);
        accelerator = requested;
        shortcutStatus = 'registered';
        revision++;
        publish();
        return snapshot();
    });
    const persistenceState = (): WindowState | null => mode === 'expanded' ? null : {
        ...clamp(expanded), isMaximized: maximized, isFullScreen: fullscreen,
    };
    window.on('close', event => {
        if (!quitting && background && shortcutStatus === 'registered') {
            event.preventDefault();
            applyMode('hidden');
        } else if (!requestQuit()) event.preventDefault();
    });
    screen.on('display-removed', visible);
    screen.on('display-metrics-changed', visible);
    window.once('closed', () => {
        cancelActivation();
        activationContext = undefined;
        if (shortcutStatus === 'registered') globalShortcut.unregister(accelerator);
        ipcMain.removeHandler('native:presentation:shortcut');
        ipcMain.removeHandler('native:presentation:get');
        ipcMain.removeHandler('native:presentation:dismiss-context');
        ipcMain.removeHandler('native:presentation:read-context-image');
        ipcMain.removeHandler('native:presentation:set');
        ipcMain.removeHandler('native:presentation:quit-guard');
        ipcMain.removeHandler('native:presentation:quit-decision');
        screen.removeListener('display-removed', visible);
        screen.removeListener('display-metrics-changed', visible);
    });
    return { persistenceState, reveal, requestQuit,
        setBackground: enabled => {
            if (enabled && shortcutStatus !== 'registered') return false;
            background = enabled;
            return true;
        },
        prepareQuit: () => { quitting = true; cancelActivation(); },
    };
}
