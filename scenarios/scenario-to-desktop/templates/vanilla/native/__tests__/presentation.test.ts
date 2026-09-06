import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { BrowserWindow } from 'electron';
const mocks = vi.hoisted(() => ({ quit: vi.fn(), activate: () => {}, register: vi.fn(), unregister: vi.fn(), handlers: new Map<string, (...args: any[]) => any>(), display: { x: 0, y: 0, width: 1920, height: 1080 }, listeners: new Map<string, () => void>() }));
vi.mock('electron', () => ({
    app: { quit: mocks.quit },
    globalShortcut: { register: mocks.register, unregister: mocks.unregister },
    ipcMain: { handle: (name: string, handler: (...args: any[]) => any) => mocks.handlers.set(name, handler), removeHandler: (name: string) => mocks.handlers.delete(name) },
    screen: { getDisplayMatching: () => ({ workArea: mocks.display }), on: (name: string, fn: (...args: any[]) => void) => mocks.listeners.set(name, fn), removeListener: (name: string) => mocks.listeners.delete(name) },
}));
import { installPresentation, type ActivationCapture } from '../presentation';
import { WindowStateManager } from '../../window-state/manager';
import type { IManagedWindow, WindowState } from '../../window-state/types';
describe('presentation shell', () => {
    beforeEach(() => { mocks.quit.mockClear(); mocks.register.mockReset().mockImplementation((_key: string, callback: () => void) => { mocks.activate = callback; return true; }); mocks.unregister.mockClear(); mocks.handlers.clear(); mocks.listeners.clear(); mocks.display = { x: 0, y: 0, width: 1920, height: 1080 }; });
    function setup(enabled = true, shortcut = false, capture?: ActivationCapture) {
        let bounds = { x: 200, y: 100, width: 1000, height: 800 };
        let close = () => {};
        const events = new Map<string, (...args: any[]) => void>();
        const webContents = { mainFrame: {}, send: vi.fn(), getURL: () => "https://app.example/chat", on: (_name: string, fn: () => void) => { fn(); }, loadURL: vi.fn(), reload: vi.fn() };
        const window = { webContents, on: (name: string, fn: (...args: any[]) => void) => { events.set(name, fn); }, removeListener: (name: string) => { events.delete(name); }, getBounds: () => bounds, getNormalBounds: () => bounds,
            setBounds: (value: typeof bounds) => { bounds = value; }, getMinimumSize: () => [400, 300],
            isMenuBarVisible: () => true, setMenuBarVisibility: vi.fn(), isResizable: () => true, isAlwaysOnTop: () => false, isDestroyed: () => false,
            maximize: vi.fn(), unmaximize: vi.fn(), isMinimized: () => false, restore: vi.fn(), hide: vi.fn(), show: vi.fn(), focus: vi.fn(), isFullScreen: () => false, isMaximized: () => false, setMinimumSize: vi.fn(), setResizable: vi.fn(),
            setAlwaysOnTop: vi.fn(), once: (_name: string, fn: () => void) => { close = fn; } };
        const presentation = installPresentation(window as unknown as BrowserWindow, enabled ? { version: capture ? 3 : shortcut ? 2 : 1, module: 'presentation', permissions: capture ? ['window.presentation', 'global-shortcut', 'desktop.context'] : shortcut ? ['window.presentation', 'global-shortcut'] : ['window.presentation'], ...(shortcut ? { activation_shortcut: 'CommandOrControl+Shift+Space' } : {}) } : null, capture);
        const event = { sender: webContents, senderFrame: webContents.mainFrame };
        return { window, event, events, presentation, close: () => close(), set: (mode: unknown) => mocks.handlers.get('native:presentation:set')!(event, mode) };
    }
    it.each(['pill', 'palette'])('restarts expanded after closing %s without saving compact bounds', async compact => {
        const { window, set, events, presentation } = setup();
        let saved: WindowState | null = null;
        const storage = { load: async () => saved, save: async (state: WindowState) => { saved = state; } };
        const display = { id: 1, ...mocks.display };
        const displayProvider = { getAllDisplays: () => [display], getPrimaryDisplay: () => display, getDisplayAtPoint: () => display };
        const manager = new WindowStateManager({ storage, displayProvider, log: () => {} });
        manager.manage(window as unknown as IManagedWindow, presentation?.persistenceState);
        set(compact);
        events.get('close')!();
        const restarted = new WindowStateManager({ storage, displayProvider, log: () => {} });
        expect(await restarted.getInitialState()).toMatchObject({ x: 200, y: 100, width: 1000, height: 800, isMaximized: false });
        set('expanded');
        window.setBounds({ x: 40, y: 50, width: 900, height: 700 });
        events.get('close')!();
        expect(saved).toMatchObject({ x: 40, y: 50, width: 900, height: 700 });
    });
    it('retains maximized intent and validates expanded bounds after display removal', () => {
        const { window, set, presentation } = setup();
        window.isMaximized = () => true;
        set('pill');
        expect(window.unmaximize).toHaveBeenCalledOnce();
        mocks.display = { x: -800, y: 0, width: 800, height: 600 };
        expect(presentation?.persistenceState()).toEqual({ x: -800, y: 0, width: 800, height: 600, isMaximized: true, isFullScreen: false });
        set('expanded');
        expect(window.maximize).toHaveBeenCalledOnce();
        expect(presentation?.persistenceState()).toBeNull();
    });
    it('hides without replacing the renderer and recovers through native activation', () => {
        const { window, set, presentation } = setup(true, true);
        expect(set('hidden')).toMatchObject({ mode: 'hidden', canHide: true });
        expect(window.hide).toHaveBeenCalledOnce();
        expect(presentation?.persistenceState()).toMatchObject({ width: 1000, height: 800 });
        mocks.activate();
        expect(window.show).toHaveBeenCalled();
        expect(window.getBounds()).toMatchObject({ width: 560, height: 480 });
        expect(set('expanded').mode).toBe('expanded');
        expect(window.getBounds()).toMatchObject({ width: 1000, height: 800 });
        expect(window.webContents.reload).not.toHaveBeenCalled();
    });
    it('refuses hidden mode when no activation path is registered', () => {
        const manual = setup();
        expect(() => manual.set('hidden')).toThrow('registered activation');
        mocks.register.mockReturnValue(false);
        const conflict = setup(true, true);
        expect(() => conflict.set('hidden')).toThrow('registered activation');
        expect(conflict.window.hide).not.toHaveBeenCalled();
    });
    it('uses the same authoritative state for native tray reveal actions', () => {
        const { window, set, event, presentation } = setup(true, true);
        set('hidden');
        presentation?.reveal('pill');
        expect(mocks.handlers.get('native:presentation:get')!(event).mode).toBe('pill');
        expect(window.getBounds()).toMatchObject({ width: 320, height: 96 });
        presentation?.reveal('expanded');
        expect(window.getBounds()).toMatchObject({ width: 1000, height: 800 });
        expect(window.focus).toHaveBeenCalledTimes(2);
        window.webContents.getURL = () => 'https://untrusted.example/';
        presentation?.reveal('palette');
        expect(window.focus).toHaveBeenCalledTimes(2);
    });
    it('keeps the renderer on configured close and permits explicit quit', () => {
        const { window, events, presentation, event } = setup(true, true);
        const preventDefault = vi.fn();
        events.get('close')!({ preventDefault });
        expect(preventDefault).not.toHaveBeenCalled();
        expect(presentation?.setBackground(true)).toBe(true);
        events.get('close')!({ preventDefault });
        expect(preventDefault).toHaveBeenCalledOnce();
        expect(window.hide).toHaveBeenCalledOnce();
        expect(mocks.handlers.get('native:presentation:get')!(event).mode).toBe('hidden');
        presentation?.prepareQuit();
        events.get('close')!({ preventDefault });
        expect(preventDefault).toHaveBeenCalledOnce();
    });
    it('refuses background without recovery and lets the user disable it', () => {
        const manual = setup();
        expect(manual.presentation?.setBackground(true)).toBe(false);
        const { events, presentation, window } = setup(true, true);
        expect(presentation?.setBackground(true)).toBe(true);
        expect(presentation?.setBackground(false)).toBe(true);
        const preventDefault = vi.fn();
        events.get('close')!({ preventDefault });
        expect(preventDefault).not.toHaveBeenCalled();
        expect(window.hide).not.toHaveBeenCalled();
    });
    it('preserves fullscreen on background close and native recovery', () => {
        const { events, presentation, window, event } = setup(true, true);
        window.isFullScreen = () => true;
        presentation?.setBackground(true);
        const preventDefault = vi.fn();
        events.get('close')!({ preventDefault });
        expect(preventDefault).toHaveBeenCalledOnce();
        expect(presentation?.persistenceState()?.isFullScreen).toBe(true);
        presentation?.reveal('palette');
        expect(mocks.handlers.get('native:presentation:get')!(event).mode).toBe('expanded');
        expect(window.getBounds()).toMatchObject({ width: 1000, height: 800 });
    });
    it('waits for a trusted matching quit decision and rejects stale replies', async () => {
        const { presentation, event, window } = setup(true, true);
        mocks.handlers.get('native:presentation:quit-guard')!(event);
        expect(presentation?.requestQuit()).toBe(false);
        const request = window.webContents.send.mock.calls.find(call => call[0] === 'native:presentation:quit-request')![1];
        expect(presentation?.requestQuit()).toBe(false);
        const decide = mocks.handlers.get('native:presentation:quit-decision')!;
        expect(() => decide(event, request + 1, 'quit')).toThrow('stale');
        expect(() => decide({ ...event, sender: {} }, request, 'quit')).toThrow('sender');
        decide(event, request, 'cancel');
        expect(mocks.quit).not.toHaveBeenCalled();
        expect(presentation?.requestQuit()).toBe(false);
        expect(() => decide(event, request, 'quit')).toThrow('stale');
        decide(event, request + 1, 'quit');
        await Promise.resolve();
        expect(mocks.quit).toHaveBeenCalledOnce();
        expect(presentation?.requestQuit()).toBe(true);
    });
    it('returns guarded Quit to the background without terminating', () => {
        const { presentation, event, window } = setup(true, true);
        mocks.handlers.get('native:presentation:quit-guard')!(event);
        presentation?.requestQuit();
        mocks.handlers.get('native:presentation:quit-decision')!(event, 1, 'background');
        expect(window.hide).toHaveBeenCalledOnce();
        expect(mocks.quit).not.toHaveBeenCalled();
    });
    it('activates the same window and publishes a newer native revision', () => {
        const { window, event, set, close } = setup(true, true);
        set('pill');
        const before = mocks.handlers.get('native:presentation:get')!(event);
        mocks.activate();
        const after = mocks.handlers.get('native:presentation:get')!(event);
        expect(after.mode).toBe('palette');
        expect(after.revision).toBeGreaterThan(before.revision);
        expect(after.shortcut.status).toBe('registered');
        expect(window.focus).toHaveBeenCalledOnce();
        expect(window.webContents.send).toHaveBeenCalledWith('native:presentation:changed', after);
        expect(window.webContents.reload).not.toHaveBeenCalled();
        close(); expect(mocks.unregister).toHaveBeenCalledWith('CommandOrControl+Shift+Space');
    });
    it('reports a refused shortcut without stealing or unregistering another binding', () => {
        mocks.register.mockReturnValue(false);
        const { event, set, close } = setup(true, true);
        expect(mocks.handlers.get('native:presentation:get')!(event).shortcut.status).toBe('unavailable');
        expect(set('palette').mode).toBe('palette');
        close(); expect(mocks.unregister).not.toHaveBeenCalled();
    });
    it('does not activate an untrusted navigated renderer', () => {
        const { window } = setup(true, true);
        window.webContents.getURL = () => 'https://untrusted.example/';
        mocks.activate(); expect(window.focus).not.toHaveBeenCalled();
    });
    it('preserves the renderer and restores expanded bounds', () => {
        const { window, set } = setup();
        expect(set('pill').mode).toBe('pill');
        expect(window.getBounds().width).toBe(320);
        expect(set('palette').mode).toBe('palette');
        expect(set('expanded').mode).toBe('expanded');
        expect(window.getBounds()).toEqual({ x: 200, y: 100, width: 1000, height: 800 });
        expect(window.setMenuBarVisibility).toHaveBeenLastCalledWith(true);
        expect(window.webContents.loadURL).not.toHaveBeenCalled();
        expect(window.webContents.reload).not.toHaveBeenCalled();
    });
    it('refuses other windows, subframes and arbitrary modes', () => {
        const { window, event, set } = setup();
        const handler = mocks.handlers.get('native:presentation:set')!;
        expect(() => handler({ ...event, sender: {} }, 'pill')).toThrow('sender');
        expect(() => handler({ ...event, senderFrame: {} }, 'pill')).toThrow('sender');
        expect(() => set({ code: 'arbitrary' })).toThrow('mode');
        window.webContents.getURL = () => 'https://untrusted.example/';
        expect(() => set('pill')).toThrow('sender');
    });
    it('handles display removal and removes handlers when closed', () => {
        const { window, set, close } = setup();
        set('palette');
        mocks.display = { x: -800, y: 0, width: 800, height: 600 };
        mocks.listeners.get('display-removed')!();
        expect(window.getBounds().x + window.getBounds().width).toBeLessThanOrEqual(0);
        close();
        expect(mocks.handlers.size).toBe(0);
        expect(mocks.listeners.size).toBe(0);
    });
    it('replaces a shortcut only after successful registration and cleans up the current binding', () => {
        const { event, close, set } = setup(true, true);
        const configure = mocks.handlers.get('native:presentation:shortcut')!;
        mocks.register.mockReturnValueOnce(false);
        expect(() => configure(event, 'Control+Alt+P')).toThrow('unavailable');
        expect(mocks.unregister).not.toHaveBeenCalled();
        expect(set('hidden').canHide).toBe(true);
        const result = configure(event, 'Control+Alt+P');
        expect(result.shortcut).toEqual({ accelerator: 'Control+Alt+P', status: 'registered' });
        expect(mocks.unregister).toHaveBeenCalledWith('CommandOrControl+Shift+Space');
        const calls = mocks.register.mock.calls.length;
        configure(event, 'Control+Alt+P');
        expect(mocks.register.mock.calls.length).toBe(calls);
        close();
        expect(mocks.unregister).toHaveBeenLastCalledWith('Control+Alt+P');
    });
    it('recovers a startup conflict but rejects unauthorized or ungoverned shortcut changes', () => {
        mocks.register.mockReturnValueOnce(false);
        const { event, set } = setup(true, true);
        const configure = mocks.handlers.get('native:presentation:shortcut')!;
        expect(() => configure({ ...event, senderFrame: {} }, 'Control+Alt+P')).toThrow('sender');
        expect(() => configure(event, 'F1')).toThrow('unsupported');
        expect(configure(event, 'Control+Alt+P').canHide).toBe(true);
        expect(mocks.unregister).not.toHaveBeenCalled();
        expect(set('hidden').mode).toBe('hidden');
        const manual = setup();
        expect(() => mocks.handlers.get('native:presentation:shortcut')!(manual.event, 'Control+Alt+P')).toThrow('unsupported');
    });
    it('captures before focusing and ignores duplicate activation while pending', async () => {
        let resolve!: (value: { contextId: string; expiresAt: number }) => void;
        const capture = vi.fn(() => new Promise<{contextId:string;expiresAt:number}>(done => { resolve = done; }));
        const { window, event } = setup(true, true, { capture });
        mocks.activate(); mocks.activate();
        expect(capture).toHaveBeenCalledOnce(); expect(window.focus).not.toHaveBeenCalled();
        resolve({ contextId: '59a6140b-a9cd-43d5-992b-2506b0177424', expiresAt: Date.now()+20000 });
        await new Promise(done => setTimeout(done, 0));
        expect(window.focus).toHaveBeenCalledOnce();
        expect(mocks.handlers.get('native:presentation:get')!(event).activation).toMatchObject({ status: 'ready', contextId: '59a6140b-a9cd-43d5-992b-2506b0177424' });
    });
    it('opens after bounded capture timeout and discards a late context', async () => {
        vi.useFakeTimers();
        try {
            let resolve!: (value: { contextId: string; expiresAt: number }) => void;
            let signal!: AbortSignal;
            const { window, event } = setup(true, true, { capture: value => { signal = value; return new Promise(done => { resolve = done; }); } });
            mocks.activate();
            await vi.advanceTimersByTimeAsync(750);
            expect(window.focus).toHaveBeenCalledOnce(); expect(signal.aborted).toBe(true);
            resolve({ contextId: '59a6140b-a9cd-43d5-992b-2506b0177424', expiresAt: Date.now()+20000 });
            await vi.advanceTimersByTimeAsync(0);
            expect(window.focus).toHaveBeenCalledOnce();
            expect(mocks.handlers.get('native:presentation:get')!(event).activation).toEqual({ status: 'unavailable' });
        } finally { vi.useRealTimers(); }
    });
    it('does not let a late capture override manual presentation or quit', async () => {
        let resolve!: (value: { contextId: string; expiresAt: number }) => void;
        const { window, set, presentation } = setup(true, true, { capture: () => new Promise(done => { resolve = done; }) });
        mocks.activate(); set('pill'); presentation?.prepareQuit();
        resolve({ contextId: '59a6140b-a9cd-43d5-992b-2506b0177424', expiresAt: Date.now()+20000 });
        await new Promise(done => setTimeout(done, 0));
        expect(window.focus).not.toHaveBeenCalled(); expect(window.getBounds().width).toBe(320);
    });
    it.each([false, true])('dismisses context while ready=%s and fences late capture without focusing', async ready => {
        let resolve!: (value: { contextId: string; expiresAt: number }) => void;
        let signal!: AbortSignal;
        const { window, event } = setup(true, true, { capture: value => { signal = value; return new Promise(done => { resolve = done; }); } });
        const discard = vi.fn(() => Promise.resolve());
        const result = { contextId: '59a6140b-a9cd-43d5-992b-2506b0177424', expiresAt: Date.now()+20000, discard };
        mocks.activate();
        if (ready) { resolve(result); await new Promise(done => setTimeout(done, 0)); }
        window.focus.mockClear();
        const dismiss = mocks.handlers.get('native:presentation:dismiss-context')!;
        await expect(dismiss({ ...event, senderFrame: {} })).rejects.toThrow('sender');
        const before = mocks.handlers.get('native:presentation:get')!(event);
        const after = await dismiss(event);
        expect(after.activation).toEqual({ status: 'unavailable' });
        expect(after.revision).toBeGreaterThan(before.revision);
        expect(after.mode).toBe(before.mode);
        expect(before.activation.discard).toBeUndefined();
        expect(signal.aborted).toBe(true);
        resolve(result); await new Promise(done => setTimeout(done, 0));
        expect(mocks.handlers.get('native:presentation:get')!(event).activation).toEqual({ status: 'unavailable' });
        expect(discard).toHaveBeenCalledOnce();
        expect(window.focus).not.toHaveBeenCalled();
        expect(window.webContents.reload).not.toHaveBeenCalled();
    });
    it('keeps a dismissed reference absent when helper deletion fails', async () => {
        const discard = vi.fn(() => Promise.reject(new Error('helper unavailable')));
        const { event } = setup(true, true, { capture: () => Promise.resolve({ contextId: '59a6140b-a9cd-43d5-992b-2506b0177424', expiresAt: Date.now()+20000, discard }) });
        mocks.activate(); await new Promise(done => setTimeout(done, 0));
        await expect(mocks.handlers.get('native:presentation:dismiss-context')!(event)).rejects.toThrow('helper unavailable');
        expect(mocks.handlers.get('native:presentation:get')!(event).activation).toEqual({ status: 'unavailable' });
        expect(discard).toHaveBeenCalledOnce();
    });
    it('keeps preview bytes out of snapshots and refuses iframe or dismissed reads', async () => {
        const contextId='59a6140b-a9cd-43d5-992b-2506b0177424', expiresAt=Date.now()+20000;
        let finish!: (value: import('../presentation').ActivationImagePreview) => void;
        const readImage=vi.fn(() => new Promise<import('../presentation').ActivationImagePreview>(resolve=>{finish=resolve;}));
        const {event}=setup(true,true,{capture:()=>Promise.resolve({contextId,expiresAt,readImage})});
        mocks.activate();await new Promise(done=>setTimeout(done,0));
        const state=mocks.handlers.get('native:presentation:get')!(event);
        expect(state.activation).toEqual({status:'ready',contextId,expiresAt,hasImage:true});
        const read=mocks.handlers.get('native:presentation:read-context-image')!;
        await expect(read({...event,senderFrame:{}})).rejects.toThrow('sender');expect(readImage).not.toHaveBeenCalled();
        const source={surface:{ownerScenario:'device-control',surfaceId:'desktop',target:{ownerScenario:'vrooli-bridge',resourceId:'host',hostNodeId:'host'}},captureId:contextId,displayId:'d',geometryRevision:'g',capturedAt:new Date().toISOString(),bounds:{x:0,y:0,width:1,height:1}};
        const preview={source,contextId,expiresAt,mimeType:'image/png' as const,dataUrl:'private',sourceBounds:{x:0,y:0,width:1,height:1}};
        const successful=read(event);finish(preview);await expect(successful).resolves.toEqual(preview);
        const pending=read(event);
        await expect(read(event)).rejects.toThrow('pending');
        await mocks.handlers.get('native:presentation:dismiss-context')!(event);
        finish({source,contextId,expiresAt,mimeType:'image/png',dataUrl:'private',sourceBounds:{x:0,y:0,width:1,height:1}});
        await expect(pending).rejects.toThrow('dismissed');
        await expect(read(event)).rejects.toThrow('unavailable');
    });
    it('registers no native handlers in vanilla apps', () => { setup(false); expect(mocks.handlers.size).toBe(0); });
});
