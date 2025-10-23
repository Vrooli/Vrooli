/**
 * Global DOM references, shared constants, and mutable state for the web-console UI.
 */

/**
 * @typedef {import("./types.d.ts").TerminalTab} TerminalTab
 * @typedef {import("./types.d.ts").AppState} AppState
 */

const DOM_ELEMENT_IDS = Object.freeze({
  sessionOverviewCount: "sessionOverviewCount",
  sessionOverviewList: "sessionOverviewList",
  sessionOverviewEmpty: "sessionOverviewEmpty",
  sessionOverviewMeta: "sessionOverviewMeta",
  sessionOverviewAlert: "sessionOverviewAlert",
  sessionOverviewRefresh: "sessionOverviewRefresh",
  detachedTabsMeta: "detachedTabsMeta",
  closeDetachedTabs: "closeDetachedTabs",
  eventFeed: "eventFeed",
  eventFeedCount: "eventFeedCount",
  eventMeta: "eventMeta",
  errorBanner: "errorBanner",
  tabList: "tabList",
  addTabBtn: "addTabBtn",
  tabAddSlot: "tabAddSlot",
  terminalHost: "terminalHost",
  layout: "mainLayout",
  drawerToggle: "drawerToggle",
  drawerClose: "drawerClose",
  drawerIndicator: "drawerIndicator",
  detailsDrawer: "detailsDrawer",
  drawerBackdrop: "drawerBackdrop",
  tabContextMenu: "tabContextMenu",
  tabContextForm: "tabContextForm",
  tabContextName: "tabContextName",
  tabContextColors: "tabContextColors",
  tabContextCancel: "tabContextCancel",
  tabContextReset: "tabContextReset",
  tabContextBackdrop: "tabContextBackdrop",
  signOutAllSessions: "signOutAllSessions",
  aiCommandInput: "aiCommandInput",
  aiGenerateBtn: "aiGenerateBtn",
  composeBtn: "composeInputBtn",
  composeDialog: "composeDialog",
  composeBackdrop: "composeBackdrop",
  composeForm: "composeForm",
  composeClose: "composeClose",
  composeCancel: "composeCancel",
  composeTextarea: "composeTextarea",
  composeAppendNewline: "composeAppendNewline",
  composeCharCount: "composeCharCount",
  composeSend: "composeSend",
  idleTimeoutToggle: "idleTimeoutToggle",
  idleTimeoutMinutes: "idleTimeoutMinutes",
});

export const elements = /** @type {Record<keyof typeof DOM_ELEMENT_IDS, HTMLElement | null>} */ (
  Object.fromEntries(
    Object.keys(DOM_ELEMENT_IDS).map((key) => [key, /** @type {HTMLElement | null} */ (null)]),
  )
);

export const shortcutButtons = /** @type {HTMLButtonElement[]} */ ([]);

/**
 * Populate cached DOM references and shortcut button list using the provided document.
 * Falls back to `globalThis.document` when available.
 * @param {Document | null | undefined} [root]
 * @returns {typeof elements}
 */
export function initializeDomElements(root) {
  const doc = root || (typeof document !== "undefined" ? document : null);

  if (!doc) {
    Object.keys(DOM_ELEMENT_IDS).forEach((key) => {
      elements[/** @type {keyof typeof DOM_ELEMENT_IDS} */ (key)] = null;
    });
    shortcutButtons.splice(0, shortcutButtons.length);
    return elements;
  }

  Object.entries(DOM_ELEMENT_IDS).forEach(([key, id]) => {
    const node = typeof doc.getElementById === "function" ? doc.getElementById(id) : null;
    elements[/** @type {keyof typeof DOM_ELEMENT_IDS} */ (key)] = /** @type {HTMLElement | null} */ (node);
  });

  const shortcuts = typeof doc.querySelectorAll === "function"
    ? /** @type {HTMLButtonElement[]} */ (
        Array.from(doc.querySelectorAll("[data-shortcut-id]"))
          .filter((node) => node instanceof HTMLButtonElement)
      )
    : [];
  shortcutButtons.splice(0, shortcutButtons.length, ...shortcuts);

  return elements;
}

export const debugFlags = (() => {
  if (typeof window === "undefined") {
    return Object.freeze({ inputTelemetry: false, metrics: false });
  }

  const globalDebug = window.__WEB_CONSOLE_DEBUG__ || {};
  const fromStorage = (() => {
    try {
      const raw = window.localStorage?.getItem("webConsoleDebug");
      return raw ? JSON.parse(raw) : {};
    } catch (_error) {
      return {};
    }
  })();

  return Object.freeze({
    inputTelemetry: Boolean(
      globalDebug.inputTelemetry || fromStorage.inputTelemetry,
    ),
    metrics: Boolean(globalDebug.metrics || fromStorage.metrics),
  });
})();

export const TAB_COLOR_OPTIONS = [
  {
    id: "sky",
    label: "Sky",
    swatch: "#38bdf8",
    styles: {
      border: "rgba(56, 189, 248, 0.28)",
      hover: "rgba(56, 189, 248, 0.45)",
      active: "rgba(56, 189, 248, 0.65)",
      selectedStart: "rgba(56, 189, 248, 0.35)",
      selectedEnd: "rgba(14, 116, 144, 0.38)",
      glow: "rgba(56, 189, 248, 0.35)",
      label: "#f8fafc",
    },
  },
  {
    id: "emerald",
    label: "Emerald",
    swatch: "#34d399",
    styles: {
      border: "rgba(52, 211, 153, 0.26)",
      hover: "rgba(16, 185, 129, 0.42)",
      active: "rgba(16, 185, 129, 0.62)",
      selectedStart: "rgba(16, 185, 129, 0.32)",
      selectedEnd: "rgba(5, 150, 105, 0.38)",
      glow: "rgba(16, 185, 129, 0.32)",
      label: "#ecfeff",
    },
  },
  {
    id: "amber",
    label: "Amber",
    swatch: "#f59e0b",
    styles: {
      border: "rgba(245, 158, 11, 0.32)",
      hover: "rgba(245, 158, 11, 0.48)",
      active: "rgba(245, 158, 11, 0.68)",
      selectedStart: "rgba(251, 191, 36, 0.35)",
      selectedEnd: "rgba(217, 119, 6, 0.42)",
      glow: "rgba(245, 158, 11, 0.35)",
      label: "#fffbeb",
    },
  },
  {
    id: "sunset",
    label: "Sunset",
    swatch: "#f97316",
    styles: {
      border: "rgba(249, 115, 22, 0.32)",
      hover: "rgba(249, 115, 22, 0.48)",
      active: "rgba(249, 115, 22, 0.7)",
      selectedStart: "rgba(251, 146, 60, 0.36)",
      selectedEnd: "rgba(194, 65, 12, 0.38)",
      glow: "rgba(249, 115, 22, 0.36)",
      label: "#fff7ed",
    },
  },
  {
    id: "violet",
    label: "Violet",
    swatch: "#a855f7",
    styles: {
      border: "rgba(168, 85, 247, 0.32)",
      hover: "rgba(139, 92, 246, 0.5)",
      active: "rgba(139, 92, 246, 0.7)",
      selectedStart: "rgba(196, 181, 253, 0.36)",
      selectedEnd: "rgba(126, 58, 242, 0.42)",
      glow: "rgba(168, 85, 247, 0.38)",
      label: "#f5f3ff",
    },
  },
  {
    id: "slate",
    label: "Slate",
    swatch: "#64748b",
    styles: {
      border: "rgba(100, 116, 139, 0.32)",
      hover: "rgba(100, 116, 139, 0.5)",
      active: "rgba(100, 116, 139, 0.7)",
      selectedStart: "rgba(148, 163, 184, 0.32)",
      selectedEnd: "rgba(71, 85, 105, 0.48)",
      glow: "rgba(100, 116, 139, 0.35)",
      label: "#f8fafc",
    },
  },
];

export const TAB_COLOR_DEFAULT = TAB_COLOR_OPTIONS[0]?.id || "sky";
export const TAB_COLOR_MAP = TAB_COLOR_OPTIONS.reduce((map, option) => {
  map[option.id] = option;
  return map;
}, {});

export const TAB_LONG_PRESS_DELAY = 550;

export const terminalDefaults = {
  convertEol: true,
  cursorBlink: true,
  rendererType: "canvas",
  fontFamily: "JetBrains Mono, SFMono-Regular, Menlo, monospace",
  fontSize: 14,
  scrollback: 400,
  theme: {
    background: "#0f172a",
    foreground: "#e2e8f0",
    cursor: "#38bdf8",
  },
};

export const INPUT_BATCH_MAX_CHARS = 64;
export const INPUT_CONTROL_FLUSH_PATTERN = /[\r\n\x03\x04\x1b\x7f]/;
export const LOCAL_ECHO_MAX_BUFFER = 2048;
export const LOCAL_ECHO_TIMEOUT_MS = 5000;
export const LOCAL_ECHO_ENABLED = false;

export const AGGREGATED_EVENT_TYPES = new Set(["ws-empty-frame"]);
export const SUPPRESSED_EVENT_LABELS = {
  "ws-empty-frame": "Empty frames suppressed",
};

export function initialSuppressedState() {
  return {
    "ws-empty-frame": 0,
  };
}

export const state = /** @type {AppState} */ ({
  tabs: /** @type {TerminalTab[]} */ ([]),
  activeTabId: null,
  bridge: null,
  workspaceSocket: null,
  workspaceReconnectTimer: null,
  drawer: {
    open: false,
    unreadCount: 0,
    previousFocus: null,
  },
  tabMenu: {
    open: false,
    tabId: null,
    selectedColor: TAB_COLOR_DEFAULT,
    anchor: { x: 0, y: 0 },
  },
  composer: {
    open: false,
    previousFocus: null,
    appendNewline: true,
  },
  sessions: {
    items: /** @type {import("./types.d.ts").SessionOverviewItem[]} */ ([]),
    loading: false,
    lastFetched: 0,
    error: /** @type {Error | null} */ (null),
    pollHandle: null,
    refreshTimer: null,
    needsRefresh: false,
    capacity: null,
  },
  workspace: {
    loading: false,
    idleTimeoutSeconds: 0,
    updatingIdleTimeout: false,
    idleControlsInitialized: false,
  },
});

let tabSequence = 0;

export function nextTabSequence() {
  tabSequence += 1;
  return tabSequence;
}

export function resetTabSequence(value = 0) {
  tabSequence = Number.isFinite(value) && value >= 0 ? value : 0;
}
