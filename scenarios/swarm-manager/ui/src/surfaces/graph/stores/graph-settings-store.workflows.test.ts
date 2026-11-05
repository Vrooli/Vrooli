import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  cloneGraphSettingsInitialState, cloneLensSettings, cloneSettingsByLens,
  createDefaultLensSettings, createDefaultSettingsByLens,
  loadPersistedSettings, savePersistedSettings, useGraphSettingsStore,
} from './graph-settings-store';

const key = 'swarm-manager.graph.settings.v5';
const original = useGraphSettingsStore.getState();

beforeEach(() => {
  localStorage.clear();
  useGraphSettingsStore.setState({ settingsByLens: createDefaultSettingsByLens(), activeLens: 'topology' });
});
afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  localStorage.clear();
  useGraphSettingsStore.setState(original);
});

describe('actual per-lens graph settings and persisted recovery', () => {
  it('keeps entity visibility independent by lens and restores the saved selection', () => {
    const store = useGraphSettingsStore.getState();
    store.setActiveLens('plan');
    store.toggleEntityFilter('backlog');
    expect(useGraphSettingsStore.getState().settingsByLens.plan.entityFilters.backlog).toBe(false);
    expect(useGraphSettingsStore.getState().settingsByLens.topology.entityFilters.backlog).toBe(true);
    store.setActiveLens('focus');
    store.setEntityFilter('goal', false);
    expect(useGraphSettingsStore.getState().settingsByLens.focus.entityFilters.goal).toBe(false);
    expect(useGraphSettingsStore.getState().settingsByLens.plan.entityFilters.goal).toBe(true);
    expect(loadPersistedSettings().plan.entityFilters.backlog).toBe(false);
    expect(loadPersistedSettings().focus.entityFilters.goal).toBe(false);
    store.setActiveLens('plan');
    store.toggleEntityFilter('backlog');
    expect(loadPersistedSettings().plan.entityFilters.backlog).toBe(true);
  });

  it('edits individual statuses without removing other statuses, and removes empty groups', () => {
    const store = useGraphSettingsStore.getState();
    store.setStatusVisibility('backlog', 'ready', false);
    store.setStatusVisibility('backlog', 'failed', true);
    store.setEntityStatusGroupVisibility('execution', ['pending', 'running'], false);
    expect(loadPersistedSettings().topology.statusFilters).toEqual({
      backlog: { ready: false, failed: true }, execution: { pending: false, running: false },
    });
    store.clearStatusFilter('backlog', 'ready');
    expect(loadPersistedSettings().topology.statusFilters.backlog).toEqual({ failed: true });
    store.clearStatusFilter('backlog', 'failed');
    expect(loadPersistedSettings().topology.statusFilters.backlog).toBeUndefined();
    expect(loadPersistedSettings().topology.statusFilters.execution).toEqual({ pending: false, running: false });
    store.clearStatusFilter('capture', 'failed');
    expect(loadPersistedSettings().topology.statusFilters.capture).toBeUndefined();
  });

  it('persists controls for the active lens and resets only the selected lens', () => {
    const store = useGraphSettingsStore.getState();
    store.setActiveLens('plan');
    store.setShowSecondaryEdges(false);
    store.setShowNavControls(true);
    store.setAutoFitOnChange(false);
    store.setHighlightActionableNodes(false);
    expect(loadPersistedSettings().plan).toMatchObject({
      showSecondaryEdges: false, showNavControls: true, autoFitOnChange: false, highlightActionableNodes: false,
    });
    store.setActiveLens('focus');
    store.setShowNavControls(true);
    store.resetLensSettings('plan');
    expect(loadPersistedSettings().plan).toEqual(createDefaultLensSettings('plan'));
    expect(loadPersistedSettings().focus.showNavControls).toBe(true);
    store.resetLensSettings();
    expect(loadPersistedSettings().focus).toEqual(createDefaultLensSettings('focus'));
    expect(loadPersistedSettings().topology).toEqual(createDefaultLensSettings('topology'));
  });

  it.each(['v2', 'v3', 'v4'])('restores legacy %s settings without changing unrelated lenses', (version) => {
    localStorage.setItem(`swarm-manager.graph.settings.${version}`, JSON.stringify({
      plan: { entityFilters: { backlog: false }, statusFilters: { ready: false }, showNavControls: true },
    }));
    const recovered = loadPersistedSettings();
    expect(recovered.plan.entityFilters.backlog).toBe(false);
    expect(recovered.plan.statusFilters.backlog?.ready).toBe(false);
    expect(recovered.plan.showNavControls).toBe(true);
    expect(recovered.focus).toEqual(createDefaultLensSettings('focus'));
    savePersistedSettings(recovered);
    expect(JSON.parse(localStorage.getItem(key)!)).toEqual(recovered);
  });

  it('prefers the current version and keeps only well-typed stored values', () => {
    localStorage.setItem('swarm-manager.graph.settings.v4', JSON.stringify({ plan: { showNavControls: true } }));
    localStorage.setItem(key, JSON.stringify({
      plan: {
        entityFilters: { backlog: false, goal: 'false', unknown: false },
        statusFilters: { backlog: { ready: false, failed: 'false' }, execution: null },
        showNavControls: false, autoFitOnChange: 'false', showSecondaryEdges: null,
      }, focus: false, topology: { statusFilters: { backlog: { ready: 'false' } } },
    }));
    const recovered = loadPersistedSettings();
    expect(recovered.plan.entityFilters.backlog).toBe(false);
    expect(recovered.plan.entityFilters.goal).toBe(true);
    expect(recovered.plan.entityFilters).not.toHaveProperty('unknown');
    expect(recovered.plan.statusFilters).toEqual({ backlog: { ready: false } });
    expect(recovered.plan).toMatchObject({ showNavControls: false, autoFitOnChange: true, showSecondaryEdges: true });
    expect(recovered.focus).toEqual(createDefaultLensSettings('focus'));
    expect(recovered.topology.statusFilters).toEqual({});
  });

  it.each(['{broken', 'null', '42'])('recovers default settings from unusable persisted data %s', (value) => {
    localStorage.setItem(key, value);
    expect(loadPersistedSettings()).toEqual(createDefaultSettingsByLens());
  });

  it('retains in-memory edits when browser persistence refuses writes', () => {
    const write = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new DOMException('blocked', 'SecurityError'); });
    expect(() => useGraphSettingsStore.getState().setShowNavControls(true)).not.toThrow();
    expect(useGraphSettingsStore.getState().settingsByLens.topology.showNavControls).toBe(true);
    expect(write).toHaveBeenCalledWith(key, expect.any(String));
    expect(localStorage.getItem(key)).toBeNull();
  });

  it('recovers defaults when browser persistence refuses reads', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new DOMException('blocked', 'SecurityError'); });
    expect(loadPersistedSettings()).toEqual(createDefaultSettingsByLens());
  });

  it('creates detached lens containers for editing a recovered snapshot', () => {
    useGraphSettingsStore.getState().setStatusVisibility('backlog', 'ready', false);
    const recovered = cloneGraphSettingsInitialState();
    const cloned = cloneSettingsByLens(recovered.settingsByLens);
    cloned.topology.entityFilters.goal = false;
    delete cloned.topology.statusFilters.backlog;
    expect(recovered.settingsByLens.topology.entityFilters.goal).toBe(true);
    expect(recovered.settingsByLens.topology.statusFilters.backlog).toEqual({ ready: false });
    const lens = cloneLensSettings(recovered.settingsByLens.plan);
    lens.entityFilters.goal = false;
    expect(recovered.settingsByLens.plan.entityFilters.goal).toBe(true);
    expect(recovered.activeLens).toBe('topology');
  });

  it('supports default settings before a browser window is installed', () => {
    vi.stubGlobal('window', undefined);
    expect(loadPersistedSettings()).toEqual(createDefaultSettingsByLens());
    expect(cloneGraphSettingsInitialState().settingsByLens).toEqual(createDefaultSettingsByLens());
    expect(() => savePersistedSettings(createDefaultSettingsByLens())).not.toThrow();
  });
});
