"use strict";
var __create = Object.create;
var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __getProtoOf = Object.getPrototypeOf;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __copyProps = (to, from, except, desc) => {
  if (from && typeof from === "object" || typeof from === "function") {
    for (let key of __getOwnPropNames(from))
      if (!__hasOwnProp.call(to, key) && key !== except)
        __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
  }
  return to;
};
var __toESM = (mod, isNodeMode, target) => (target = mod != null ? __create(__getProtoOf(mod)) : {}, __copyProps(
  // If the importer is in node compatibility mode or this is not an ESM
  // file that has been converted to a CommonJS file using a Babel-
  // compatible transform (i.e. "__esModule" has not been set), then set
  // "default" to the CommonJS "module.exports" for node compatibility.
  isNodeMode || !mod || !mod.__esModule ? __defProp(target, "default", { value: mod, enumerable: true }) : target,
  mod
));

// src/audio-bisect.ts
var import_rebrowser_playwright2 = require("rebrowser-playwright");

// src/session/cdp-session.ts
var cdpSessionCache = /* @__PURE__ */ new WeakMap();
async function getCachedCDPSession(page) {
  let session = cdpSessionCache.get(page);
  if (!session) {
    session = await page.context().newCDPSession(page);
    cdpSessionCache.set(page, session);
  }
  return session;
}

// src/utils/logger.ts
var import_winston = __toESM(require("winston"));
var LogContext = {
  SERVER: "server",
  SESSION: "session",
  BROWSER: "browser",
  INSTRUCTION: "instruction",
  RECORDING: "recording",
  CLEANUP: "cleanup",
  TELEMETRY: "telemetry",
  HEALTH: "health",
  /**
   * Script injection diagnostics.
   * Logs related to injecting recording scripts into pages.
   * Use for debugging injection failures or verification.
   */
  INJECTION: "injection",
  /**
   * Event flow tracing.
   * Logs related to events flowing from browser to Node.js.
   * Use for debugging event communication issues.
   */
  EVENT_FLOW: "event-flow",
  /**
   * Configuration changes.
   * Logs related to runtime configuration updates.
   */
  CONFIG: "config"
};
var loggerInstance = import_winston.default.createLogger({
  level: process.env.NODE_ENV === "test" ? "silent" : "info",
  silent: process.env.NODE_ENV === "test",
  format: import_winston.default.format.combine(
    import_winston.default.format.timestamp(),
    import_winston.default.format.errors({ stack: true }),
    import_winston.default.format.json()
  ),
  transports: [new import_winston.default.transports.Console()]
});
var logger = new Proxy({}, {
  get(_target, prop) {
    return loggerInstance[prop];
  }
});
function scopedLog(context, event) {
  return `${context}: ${event}`;
}
function createNoOpLogger() {
  return import_winston.default.createLogger({
    level: "silent",
    silent: true,
    transports: []
  });
}

// src/utils/metrics.ts
var import_prom_client = require("prom-client");
var Metrics = class {
  registry;
  sessionCount;
  instructionDuration;
  instructionErrors;
  screenshotSize;
  sessionDuration;
  cleanupFailures;
  /** Total recorded actions across all sessions */
  recordingActionsTotal;
  /** Active recording sessions */
  recordingSessionsActive;
  /** Telemetry capture failures (screenshot, DOM) */
  telemetryFailures;
  /** Recording callback streaming failures */
  recordingCallbackFailures;
  /** Circuit breaker state changes */
  circuitBreakerStateChanges;
  // Performance debug mode metrics
  /** Frame capture latency histogram (screenshot time in ms) */
  frameCaptureLatency;
  /** Frame end-to-end latency histogram (driver-side total in ms) */
  frameE2ELatency;
  /** Frame skip counter (frames not sent due to unchanged content or timeout) */
  frameSkipCount;
  constructor() {
    this.registry = new import_prom_client.Registry();
    this.sessionCount = new import_prom_client.Gauge({
      name: "playwright_driver_sessions",
      help: "Current number of sessions by state (active, total, idle)",
      labelNames: ["state"],
      registers: [this.registry]
    });
    this.instructionDuration = new import_prom_client.Histogram({
      name: "playwright_driver_instruction_duration_ms",
      help: "Instruction execution duration in milliseconds",
      labelNames: ["type", "success"],
      buckets: [10, 50, 100, 250, 500, 1e3, 2500, 5e3, 1e4],
      registers: [this.registry]
    });
    this.instructionErrors = new import_prom_client.Counter({
      name: "playwright_driver_instruction_errors_total",
      help: "Total number of instruction errors by type and error kind",
      labelNames: ["type", "error_kind"],
      registers: [this.registry]
    });
    this.screenshotSize = new import_prom_client.Histogram({
      name: "playwright_driver_screenshot_size_bytes",
      help: "Screenshot size in bytes",
      buckets: [1e4, 5e4, 1e5, 25e4, 5e5, 1e6],
      registers: [this.registry]
    });
    this.sessionDuration = new import_prom_client.Histogram({
      name: "playwright_driver_session_duration_ms",
      help: "Session lifetime duration in milliseconds",
      buckets: [1e3, 5e3, 1e4, 3e4, 6e4, 3e5, 6e5],
      registers: [this.registry]
    });
    this.cleanupFailures = new import_prom_client.Counter({
      name: "playwright_driver_cleanup_failures_total",
      help: "Total number of cleanup failures by operation type (page_close, context_close, tracing_stop, browser_close, recording_stop)",
      labelNames: ["operation"],
      registers: [this.registry]
    });
    this.recordingActionsTotal = new import_prom_client.Counter({
      name: "playwright_driver_recording_actions_total",
      help: "Total number of user actions recorded across all sessions",
      registers: [this.registry]
    });
    this.recordingSessionsActive = new import_prom_client.Gauge({
      name: "playwright_driver_recording_sessions_active",
      help: "Current number of sessions with active recording",
      registers: [this.registry]
    });
    this.telemetryFailures = new import_prom_client.Counter({
      name: "playwright_driver_telemetry_failures_total",
      help: "Total number of telemetry capture failures (screenshot, dom)",
      labelNames: ["type"],
      registers: [this.registry]
    });
    this.recordingCallbackFailures = new import_prom_client.Counter({
      name: "playwright_driver_recording_callback_failures_total",
      help: "Total number of recording callback streaming failures by reason (network, timeout, http_error)",
      labelNames: ["reason"],
      registers: [this.registry]
    });
    this.circuitBreakerStateChanges = new import_prom_client.Counter({
      name: "playwright_driver_circuit_breaker_state_changes_total",
      help: "Circuit breaker state changes for recording callbacks (opened, closed)",
      labelNames: ["session_id", "state"],
      registers: [this.registry]
    });
    this.frameCaptureLatency = new import_prom_client.Histogram({
      name: "playwright_driver_frame_capture_latency_ms",
      help: "Time to capture screenshot from Playwright (page.screenshot()) in milliseconds",
      labelNames: ["session_id"],
      buckets: [10, 25, 50, 75, 100, 150, 200, 300, 500, 1e3],
      registers: [this.registry]
    });
    this.frameE2ELatency = new import_prom_client.Histogram({
      name: "playwright_driver_frame_e2e_latency_ms",
      help: "Driver-side end-to-end frame processing time (capture + compare + send) in milliseconds",
      labelNames: ["session_id"],
      buckets: [20, 50, 100, 150, 200, 300, 500, 750, 1e3, 2e3],
      registers: [this.registry]
    });
    this.frameSkipCount = new import_prom_client.Counter({
      name: "playwright_driver_frame_skip_total",
      help: "Total frames skipped (not sent to API) by reason",
      labelNames: ["session_id", "reason"],
      registers: [this.registry]
    });
  }
  getRegistry() {
    return this.registry;
  }
  async getMetrics() {
    return this.registry.metrics();
  }
};
var metrics = new Metrics();
var noOp = () => {
};
var noOpGauge = { inc: noOp, dec: noOp, set: noOp };
var noOpCounter = { inc: noOp };
var noOpHistogram = { observe: noOp };
function createNoOpMetrics() {
  return {
    sessionCount: noOpGauge,
    instructionDuration: noOpHistogram,
    instructionErrors: noOpCounter,
    screenshotSize: noOpHistogram,
    sessionDuration: noOpHistogram,
    cleanupFailures: noOpCounter,
    recordingActionsTotal: noOpCounter,
    recordingSessionsActive: noOpGauge,
    telemetryFailures: noOpCounter,
    recordingCallbackFailures: noOpCounter,
    circuitBreakerStateChanges: noOpCounter,
    frameCaptureLatency: noOpHistogram,
    frameE2ELatency: noOpHistogram,
    frameSkipCount: noOpCounter
  };
}

// node_modules/@vrooli/proto-types/browser-automation-studio/v1/execution/driver_pb.ts
var import_codegenv29 = require("@bufbuild/protobuf/codegenv2");

// node_modules/@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb.ts
var import_codegenv26 = require("@bufbuild/protobuf/codegenv2");

// node_modules/@vrooli/proto-types/browser-automation-studio/v1/base/geometry_pb.ts
var import_codegenv2 = require("@bufbuild/protobuf/codegenv2");

// node_modules/@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb.ts
var import_codegenv23 = require("@bufbuild/protobuf/codegenv2");

// node_modules/@vrooli/proto-types/common/v1/types_pb.ts
var import_codegenv22 = require("@bufbuild/protobuf/codegenv2");
var import_wkt = require("@bufbuild/protobuf/wkt");

// node_modules/@vrooli/proto-types/browser-automation-studio/v1/domain/selectors_pb.ts
var import_codegenv24 = require("@bufbuild/protobuf/codegenv2");

// node_modules/@vrooli/proto-types/buf/validate/validate_pb.ts
var import_codegenv25 = require("@bufbuild/protobuf/codegenv2");
var import_wkt2 = require("@bufbuild/protobuf/wkt");

// node_modules/@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb.ts
var import_wkt3 = require("@bufbuild/protobuf/wkt");
var ActionType = /* @__PURE__ */ ((ActionType3) => {
  ActionType3[ActionType3["UNSPECIFIED"] = 0] = "UNSPECIFIED";
  ActionType3[ActionType3["NAVIGATE"] = 1] = "NAVIGATE";
  ActionType3[ActionType3["CLICK"] = 2] = "CLICK";
  ActionType3[ActionType3["INPUT"] = 3] = "INPUT";
  ActionType3[ActionType3["WAIT"] = 4] = "WAIT";
  ActionType3[ActionType3["ASSERT"] = 5] = "ASSERT";
  ActionType3[ActionType3["SCROLL"] = 6] = "SCROLL";
  ActionType3[ActionType3["SELECT"] = 7] = "SELECT";
  ActionType3[ActionType3["EVALUATE"] = 8] = "EVALUATE";
  ActionType3[ActionType3["KEYBOARD"] = 9] = "KEYBOARD";
  ActionType3[ActionType3["HOVER"] = 10] = "HOVER";
  ActionType3[ActionType3["SCREENSHOT"] = 11] = "SCREENSHOT";
  ActionType3[ActionType3["FOCUS"] = 12] = "FOCUS";
  ActionType3[ActionType3["BLUR"] = 13] = "BLUR";
  ActionType3[ActionType3["SUBFLOW"] = 14] = "SUBFLOW";
  ActionType3[ActionType3["EXTRACT"] = 15] = "EXTRACT";
  ActionType3[ActionType3["UPLOAD_FILE"] = 16] = "UPLOAD_FILE";
  ActionType3[ActionType3["DOWNLOAD"] = 17] = "DOWNLOAD";
  ActionType3[ActionType3["FRAME_SWITCH"] = 18] = "FRAME_SWITCH";
  ActionType3[ActionType3["TAB_SWITCH"] = 19] = "TAB_SWITCH";
  ActionType3[ActionType3["COOKIE_STORAGE"] = 20] = "COOKIE_STORAGE";
  ActionType3[ActionType3["SHORTCUT"] = 21] = "SHORTCUT";
  ActionType3[ActionType3["DRAG_DROP"] = 22] = "DRAG_DROP";
  ActionType3[ActionType3["GESTURE"] = 23] = "GESTURE";
  ActionType3[ActionType3["NETWORK_MOCK"] = 24] = "NETWORK_MOCK";
  ActionType3[ActionType3["ROTATE"] = 25] = "ROTATE";
  ActionType3[ActionType3["SET_VARIABLE"] = 26] = "SET_VARIABLE";
  ActionType3[ActionType3["LOOP"] = 27] = "LOOP";
  ActionType3[ActionType3["CONDITIONAL"] = 28] = "CONDITIONAL";
  return ActionType3;
})(ActionType || {});

// node_modules/@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb.ts
var import_codegenv28 = require("@bufbuild/protobuf/codegenv2");

// node_modules/@vrooli/proto-types/browser-automation-studio/v1/domain/telemetry_pb.ts
var import_codegenv27 = require("@bufbuild/protobuf/codegenv2");
var import_wkt4 = require("@bufbuild/protobuf/wkt");

// node_modules/@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb.ts
var import_wkt5 = require("@bufbuild/protobuf/wkt");

// node_modules/@vrooli/proto-types/browser-automation-studio/v1/execution/driver_pb.ts
var import_wkt6 = require("@bufbuild/protobuf/wkt");

// src/proto/utils.ts
var import_protobuf = require("@bufbuild/protobuf");
var import_wkt7 = require("@bufbuild/protobuf/wkt");

// src/proto/recording.ts
var import_protobuf2 = require("@bufbuild/protobuf");
var import_wkt8 = require("@bufbuild/protobuf/wkt");
var import_uuid = require("uuid");

// src/proto/action-type-utils.ts
var ACTION_TYPE_STRING_MAP = /* @__PURE__ */ new Map([
  [1 /* NAVIGATE */, "navigate"],
  [2 /* CLICK */, "click"],
  [3 /* INPUT */, "input"],
  [4 /* WAIT */, "wait"],
  [5 /* ASSERT */, "assert"],
  [6 /* SCROLL */, "scroll"],
  [7 /* SELECT */, "select"],
  [8 /* EVALUATE */, "evaluate"],
  [9 /* KEYBOARD */, "keyboard"],
  [10 /* HOVER */, "hover"],
  [11 /* SCREENSHOT */, "screenshot"],
  [12 /* FOCUS */, "focus"],
  [13 /* BLUR */, "blur"],
  [14 /* SUBFLOW */, "subflow"],
  [15 /* EXTRACT */, "extract"],
  [16 /* UPLOAD_FILE */, "uploadfile"],
  [17 /* DOWNLOAD */, "download"],
  [18 /* FRAME_SWITCH */, "frame-switch"],
  [19 /* TAB_SWITCH */, "tab-switch"],
  [20 /* COOKIE_STORAGE */, "cookie-storage"],
  [21 /* SHORTCUT */, "shortcut"],
  [22 /* DRAG_DROP */, "drag-drop"],
  [23 /* GESTURE */, "gesture"],
  [24 /* NETWORK_MOCK */, "network-mock"],
  [25 /* ROTATE */, "rotate"]
]);
var STRING_TO_ACTION_TYPE_MAP = /* @__PURE__ */ new Map([
  // Core action types (canonical names)
  ["navigate", 1 /* NAVIGATE */],
  ["click", 2 /* CLICK */],
  ["input", 3 /* INPUT */],
  ["wait", 4 /* WAIT */],
  ["assert", 5 /* ASSERT */],
  ["scroll", 6 /* SCROLL */],
  ["select", 7 /* SELECT */],
  ["evaluate", 8 /* EVALUATE */],
  ["keyboard", 9 /* KEYBOARD */],
  ["hover", 10 /* HOVER */],
  ["screenshot", 11 /* SCREENSHOT */],
  ["focus", 12 /* FOCUS */],
  ["blur", 13 /* BLUR */],
  ["subflow", 14 /* SUBFLOW */],
  ["extract", 15 /* EXTRACT */],
  ["download", 17 /* DOWNLOAD */],
  ["shortcut", 21 /* SHORTCUT */],
  // Upload file variations
  ["uploadfile", 16 /* UPLOAD_FILE */],
  ["upload", 16 /* UPLOAD_FILE */],
  // Frame switch variations
  ["frame-switch", 18 /* FRAME_SWITCH */],
  ["frameswitch", 18 /* FRAME_SWITCH */],
  // Tab switch variations
  ["tab-switch", 19 /* TAB_SWITCH */],
  ["tabswitch", 19 /* TAB_SWITCH */],
  ["tab", 19 /* TAB_SWITCH */],
  ["tabs", 19 /* TAB_SWITCH */],
  // Cookie storage variations
  ["cookie-storage", 20 /* COOKIE_STORAGE */],
  ["cookiestorage", 20 /* COOKIE_STORAGE */],
  // Drag drop variations
  ["drag-drop", 22 /* DRAG_DROP */],
  ["dragdrop", 22 /* DRAG_DROP */],
  ["drag", 22 /* DRAG_DROP */],
  // Gesture variations
  ["gesture", 23 /* GESTURE */],
  ["swipe", 23 /* GESTURE */],
  ["pinch", 23 /* GESTURE */],
  ["zoom", 23 /* GESTURE */],
  // Network mock variations
  ["network-mock", 24 /* NETWORK_MOCK */],
  ["networkmock", 24 /* NETWORK_MOCK */],
  ["network", 24 /* NETWORK_MOCK */],
  ["mock", 24 /* NETWORK_MOCK */],
  ["intercept", 24 /* NETWORK_MOCK */],
  // Rotate variations
  ["rotate", 25 /* ROTATE */],
  ["orientation", 25 /* ROTATE */],
  ["device", 25 /* ROTATE */]
]);
var ACTION_TYPE_MAP = {
  // === Core action types (direct 1:1 mapping) ===
  click: 2 /* CLICK */,
  navigate: 1 /* NAVIGATE */,
  scroll: 6 /* SCROLL */,
  select: 7 /* SELECT */,
  hover: 10 /* HOVER */,
  focus: 12 /* FOCUS */,
  blur: 13 /* BLUR */,
  wait: 4 /* WAIT */,
  assert: 5 /* ASSERT */,
  screenshot: 11 /* SCREENSHOT */,
  evaluate: 8 /* EVALUATE */,
  // === Input variations ===
  type: 3 /* INPUT */,
  input: 3 /* INPUT */,
  // === Keyboard variations ===
  keyboard: 9 /* KEYBOARD */,
  keypress: 9 /* KEYBOARD */,
  keydown: 9 /* KEYBOARD */,
  keyup: 9 /* KEYBOARD */,
  // === Browser event aliases ===
  // These are raw DOM event names that should map to our canonical types
  change: 7 /* SELECT */,
  // <select> change events
  mousedown: 2 /* CLICK */,
  // Mouse button press
  mouseup: 2 /* CLICK */,
  // Mouse button release
  dblclick: 2 /* CLICK */
  // Double-click events
};
var SELECTOR_OPTIONAL_ACTIONS = /* @__PURE__ */ new Set([
  6 /* SCROLL */,
  1 /* NAVIGATE */,
  4 /* WAIT */,
  9 /* KEYBOARD */,
  11 /* SCREENSHOT */,
  8 /* EVALUATE */
]);
function actionTypeToString(actionType) {
  return ACTION_TYPE_STRING_MAP.get(actionType) ?? "unknown";
}

// src/proto/recording.ts
var SELECTOR_TYPE_MAP = {
  css: 1 /* CSS */,
  xpath: 2 /* XPATH */,
  id: 3 /* ID */,
  "data-testid": 4 /* DATA_TESTID */,
  "data-attr": 4 /* DATA_TESTID */,
  aria: 5 /* ARIA */,
  text: 6 /* TEXT */,
  role: 7 /* ROLE */,
  placeholder: 8 /* PLACEHOLDER */,
  "alt-text": 9 /* ALT_TEXT */,
  title: 10 /* TITLE */
};
var MOUSE_BUTTON_MAP = {
  left: 1 /* LEFT */,
  right: 2 /* RIGHT */,
  middle: 3 /* MIDDLE */
};
var MODIFIER_MAP = {
  ctrl: 1 /* CTRL */,
  shift: 2 /* SHIFT */,
  alt: 3 /* ALT */,
  meta: 4 /* META */
};

// src/utils/errors.ts
var import_zod = require("zod");
var PlaywrightDriverError = class extends Error {
  constructor(message, code, kind = 1 /* ENGINE */, retryable = false, details) {
    super(message);
    this.code = code;
    this.kind = kind;
    this.retryable = retryable;
    this.details = details;
    this.name = "PlaywrightDriverError";
  }
};
var UnsupportedInstructionError = class extends PlaywrightDriverError {
  constructor(type) {
    super(
      `Unsupported instruction type: ${type}`,
      "UNSUPPORTED_INSTRUCTION",
      3 /* ORCHESTRATION */,
      false,
      { type }
    );
    this.name = "UnsupportedInstructionError";
  }
};

// src/utils/validation.ts
function normalizeInstructionType(type) {
  if (!type || typeof type !== "string") {
    throw new Error(`Instruction type is required but got: ${type}`);
  }
  return type.toLowerCase().trim();
}
var lastHrTime = process.hrtime.bigint();

// src/service-worker/controller.ts
var ServiceWorkerController = class {
  sessionId;
  registrations = /* @__PURE__ */ new Map();
  versions = /* @__PURE__ */ new Map();
  enabled = false;
  cdpSession = null;
  control;
  constructor(sessionId, control) {
    this.sessionId = sessionId;
    this.control = control;
  }
  extractWorkerErrorMessage(event) {
    if (event && typeof event === "object") {
      const record = event;
      const errorMessage = record.errorMessage;
      if (typeof errorMessage === "string") {
        return errorMessage;
      }
      if (errorMessage && typeof errorMessage === "object") {
        const nested = errorMessage;
        if (typeof nested.message === "string") {
          return nested.message;
        }
      }
    }
    return "unknown error";
  }
  /**
   * Initialize SW monitoring via CDP.
   */
  async enable(page) {
    if (this.enabled) return;
    try {
      this.cdpSession = await getCachedCDPSession(page);
      await this.cdpSession.send("ServiceWorker.enable");
      this.enabled = true;
      this.cdpSession.on("ServiceWorker.workerRegistrationUpdated", (event) => {
        this.handleRegistrationUpdate(event.registrations);
      });
      this.cdpSession.on("ServiceWorker.workerVersionUpdated", (event) => {
        this.handleVersionUpdate(event.versions);
      });
      this.cdpSession.on("ServiceWorker.workerErrorReported", (event) => {
        logger.warn(scopedLog(LogContext.SESSION, "service worker error"), {
          sessionId: this.sessionId,
          errorMessage: this.extractWorkerErrorMessage(event)
        });
      });
      logger.debug(scopedLog(LogContext.SESSION, "service worker monitoring enabled"), {
        sessionId: this.sessionId,
        mode: this.control.mode
      });
    } catch (error) {
      logger.warn(scopedLog(LogContext.SESSION, "failed to enable SW monitoring"), {
        sessionId: this.sessionId,
        error: error instanceof Error ? error.message : String(error)
      });
    }
  }
  /**
   * Disable SW monitoring.
   */
  async disable() {
    if (!this.enabled || !this.cdpSession) return;
    try {
      await this.cdpSession.send("ServiceWorker.disable");
      this.enabled = false;
      this.registrations.clear();
      this.versions.clear();
      logger.debug(scopedLog(LogContext.SESSION, "service worker monitoring disabled"), {
        sessionId: this.sessionId
      });
    } catch {
    }
  }
  /**
   * Unregister a specific service worker by scope URL.
   */
  async unregister(scopeURL) {
    if (!this.cdpSession) return false;
    try {
      await this.cdpSession.send("ServiceWorker.unregister", { scopeURL });
      logger.info(scopedLog(LogContext.SESSION, "service worker unregistered"), {
        sessionId: this.sessionId,
        scopeURL
      });
      return true;
    } catch (error) {
      logger.warn(scopedLog(LogContext.SESSION, "failed to unregister SW"), {
        sessionId: this.sessionId,
        scopeURL,
        error: error instanceof Error ? error.message : String(error)
      });
      return false;
    }
  }
  /**
   * Unregister all service workers.
   */
  async unregisterAll() {
    let count = 0;
    for (const reg of this.registrations.values()) {
      if (!reg.isDeleted) {
        const success = await this.unregister(reg.scopeURL);
        if (success) count++;
      }
    }
    if (count > 0) {
      logger.info(scopedLog(LogContext.SESSION, "all service workers unregistered"), {
        sessionId: this.sessionId,
        count
      });
    }
    return count;
  }
  /**
   * Stop all running service workers.
   */
  async stopAll() {
    if (!this.cdpSession) return;
    try {
      await this.cdpSession.send("ServiceWorker.stopAllWorkers");
      logger.info(scopedLog(LogContext.SESSION, "all service workers stopped"), {
        sessionId: this.sessionId
      });
    } catch (error) {
      logger.warn(scopedLog(LogContext.SESSION, "failed to stop all SWs"), {
        sessionId: this.sessionId,
        error: error instanceof Error ? error.message : String(error)
      });
    }
  }
  /**
   * Get list of all registered service workers.
   */
  getWorkers() {
    const workers = [];
    for (const reg of this.registrations.values()) {
      if (reg.isDeleted) continue;
      const version = [...this.versions.values()].find(
        (v) => v.registrationId === reg.registrationId
      );
      workers.push({
        registrationId: reg.registrationId,
        scopeURL: reg.scopeURL,
        scriptURL: version?.scriptURL || "",
        status: this.mapStatus(version?.runningStatus),
        versionId: version?.versionId
      });
    }
    return workers;
  }
  /**
   * Get current control settings.
   */
  getControl() {
    return this.control;
  }
  /**
   * Check if a domain should have SWs blocked.
   */
  shouldBlockDomain(domain) {
    if (this.control.domainOverrides) {
      for (const override of this.control.domainOverrides) {
        if (this.matchDomain(domain, override.domain)) {
          return override.mode === "block";
        }
      }
    }
    if (this.control.mode === "block-on-domain" && this.control.blockedDomains) {
      return this.control.blockedDomains.some((d) => this.matchDomain(domain, d));
    }
    return this.control.mode === "block";
  }
  /**
   * Setup script injection to block SW registration for specific domains.
   */
  async setupBlockingForContext(context) {
    if (this.control.mode === "allow" && !this.control.domainOverrides?.length) {
      return;
    }
    const blockScript = this.generateBlockingScript();
    await context.addInitScript(blockScript);
    logger.debug(scopedLog(LogContext.SESSION, "SW blocking script injected"), {
      sessionId: this.sessionId,
      mode: this.control.mode,
      blockedDomains: this.control.blockedDomains,
      overrideCount: this.control.domainOverrides?.length || 0
    });
  }
  // Private helpers
  handleRegistrationUpdate(registrations2) {
    for (const reg of registrations2) {
      if (reg.isDeleted) {
        this.registrations.delete(reg.registrationId);
        logger.debug(scopedLog(LogContext.SESSION, "SW registration removed"), {
          sessionId: this.sessionId,
          scopeURL: reg.scopeURL
        });
      } else {
        this.registrations.set(reg.registrationId, reg);
        try {
          const url = new URL(reg.scopeURL);
          if (this.shouldBlockDomain(url.hostname)) {
            logger.debug(scopedLog(LogContext.SESSION, "auto-unregistering blocked SW"), {
              sessionId: this.sessionId,
              scopeURL: reg.scopeURL,
              hostname: url.hostname
            });
            void this.unregister(reg.scopeURL);
          }
        } catch {
        }
      }
    }
  }
  handleVersionUpdate(versions) {
    for (const ver of versions) {
      this.versions.set(ver.versionId, ver);
    }
  }
  mapStatus(cdpStatus) {
    switch (cdpStatus) {
      case "running":
        return "running";
      case "starting":
        return "activating";
      case "stopped":
      case "stopping":
      default:
        return "stopped";
    }
  }
  /**
   * Match a domain against a pattern.
   * Supports wildcard patterns like "*.google.com"
   */
  matchDomain(actual, pattern) {
    if (pattern.startsWith("*.")) {
      const suffix = pattern.slice(2);
      return actual === suffix || actual.endsWith("." + suffix);
    }
    return actual === pattern;
  }
  /**
   * Generate the blocking script to inject into pages.
   * This script overrides navigator.serviceWorker.register to block
   * registrations for configured domains.
   */
  generateBlockingScript() {
    const blockedDomains = this.control.blockedDomains || [];
    const overrides = this.control.domainOverrides || [];
    const mode = this.control.mode;
    return `
      (function() {
        const originalRegister = navigator.serviceWorker?.register;
        if (!originalRegister) return;

        const blockedDomains = ${JSON.stringify(blockedDomains)};
        const overrides = ${JSON.stringify(overrides)};
        const mode = '${mode}';

        function matchDomain(actual, pattern) {
          if (pattern.startsWith('*.')) {
            const suffix = pattern.slice(2);
            return actual === suffix || actual.endsWith('.' + suffix);
          }
          return actual === pattern;
        }

        function shouldBlock(hostname) {
          // Check overrides first
          for (const override of overrides) {
            if (matchDomain(hostname, override.domain)) {
              return override.mode === 'block';
            }
          }
          // Check blockedDomains
          if (mode === 'block-on-domain') {
            return blockedDomains.some(d => matchDomain(hostname, d));
          }
          return mode === 'block';
        }

        navigator.serviceWorker.register = function(scriptURL, options) {
          const hostname = window.location.hostname;
          if (shouldBlock(hostname)) {
            console.warn('[playwright-driver] SW registration blocked for:', hostname);
            return Promise.reject(new DOMException('Service worker blocked by automation driver', 'SecurityError'));
          }
          return originalRegister.call(navigator.serviceWorker, scriptURL, options);
        };
      })();
    `;
  }
};

// src/recording/validation/selector-config.ts
var TEST_ID_ATTRIBUTES = [
  "data-testid",
  "data-test-id",
  "data-test",
  "data-cy",
  "data-qa"
];
var UNSTABLE_CLASS_PATTERNS = [
  { pattern: "^css-[a-z0-9]+$", flags: "i" },
  // CSS-in-JS (Emotion, etc.)
  { pattern: "^sc-[a-zA-Z]+$", flags: "" },
  // styled-components
  { pattern: "^_[a-zA-Z0-9]+$", flags: "" },
  // CSS modules
  { pattern: "^[a-zA-Z]+-[0-9]+$", flags: "" },
  // Generic hash patterns
  { pattern: "^jsx-[a-z0-9]+$", flags: "i" },
  // Next.js styled-jsx
  { pattern: "^svelte-[a-z0-9]+$", flags: "i" },
  // Svelte scoped styles
  { pattern: "^v-[a-z0-9]+$", flags: "i" }
  // Vue scoped styles
];
var DYNAMIC_ID_PATTERNS = [
  { pattern: "^[a-f0-9]{8,}$", flags: "i" },
  // Hex hash
  { pattern: "^\\d+$", flags: "" },
  // Pure numbers
  { pattern: ":r[0-9]+:", flags: "" },
  // React auto IDs
  { pattern: "^:r", flags: "" },
  // React 18 useId
  { pattern: "^ember\\d+$", flags: "" },
  // Ember
  { pattern: "^___gatsby", flags: "" },
  // Gatsby
  { pattern: "_\\d{10,}", flags: "" },
  // Timestamp-based
  { pattern: "^[a-z]+_[a-z0-9]{6,}$", flags: "i" }
  // Component_hash pattern
];
var SEMANTIC_CLASS_PATTERNS = [
  { pattern: "^btn-", flags: "" },
  { pattern: "^button-", flags: "" },
  { pattern: "^nav-", flags: "" },
  { pattern: "^header-", flags: "" },
  { pattern: "^footer-", flags: "" },
  { pattern: "^card-", flags: "" },
  { pattern: "^form-", flags: "" },
  { pattern: "^input-", flags: "" },
  { pattern: "^modal-", flags: "" },
  { pattern: "^sidebar-", flags: "" },
  { pattern: "^menu-", flags: "" }
];
var TEXT_CONTENT_TAGS = [
  "button",
  "a",
  "span",
  "div",
  "p",
  "h1",
  "h2",
  "h3",
  "h4",
  "h5",
  "h6",
  "label",
  "li",
  "td",
  "th"
];
var SELECTOR_DEFAULTS = {
  /**
   * Maximum depth for CSS path traversal.
   * Configurable via RECORDING_MAX_CSS_DEPTH (default: 5, range: 2-10)
   * Lower = shorter selectors but may be less unique.
   * Higher = more unique selectors but may be fragile.
   */
  maxCssDepth: 5,
  /**
   * Whether to include XPath as fallback strategy.
   * Configurable via RECORDING_INCLUDE_XPATH (default: true)
   * Set to false for CSS-only selector generation.
   */
  includeXPath: true,
  /** Whether to prefer data-testid selectors */
  preferTestIds: true,
  /**
   * Minimum confidence threshold to include a candidate (default).
   * Configurable via RECORDING_MIN_SELECTOR_CONFIDENCE (default: 0.3, range: 0-1)
   * Higher = stricter, fewer candidates. Lower = more candidates.
   */
  minConfidence: 0.3,
  /** Maximum text length to use in selectors */
  maxTextLength: 100,
  /** Maximum text length for selector text content */
  selectorTextMaxLength: 30,
  /** Minimum text length to consider for text-based selectors */
  minTextLength: 3,
  /** Maximum text length to consider for text-based selectors */
  maxTextLengthForSelector: 50
};
var CONFIDENCE_SCORES = {
  /** data-testid attribute - explicitly stable */
  dataTestId: 0.98,
  /** Unique ID - very stable unless dynamic */
  id: 0.95,
  /** Dynamic ID - less stable */
  idDynamic: 0.6,
  /** ARIA label - semantically stable */
  ariaLabel: 0.85,
  /** ARIA labelledby */
  ariaLabelledby: 0.8,
  /** ARIA describedby */
  ariaDescribedby: 0.75,
  /** Other data attributes */
  dataAttr: 0.7,
  /** CSS path baseline */
  cssPath: 0.65,
  /** CSS with nth-child (less stable) */
  cssNthChild: 0.5,
  /** Text-based selector (longer text) */
  textLong: 0.6,
  /** Text-based selector (shorter text) */
  textShort: 0.55,
  /** XPath text-based */
  xpathText: 0.55,
  /** XPath positional */
  xpathPositional: 0.4
};
var SPECIFICITY_SCORES = {
  dataTestId: 100,
  id: 95,
  ariaLabel: 80,
  ariaLabelledby: 75,
  ariaDescribedby: 70,
  dataAttr: 65,
  cssPath: 60,
  text: 50,
  cssNthChild: 40,
  xpathText: 35,
  xpathPositional: 25
};
var RECORDING_DEBOUNCE = {
  /**
   * Debounce for input events to batch keystrokes.
   * Configurable via RECORDING_INPUT_DEBOUNCE_MS (default: 500, range: 50-2000)
   */
  input: 500,
  /**
   * Debounce for scroll events.
   * Configurable via RECORDING_SCROLL_DEBOUNCE_MS (default: 150, range: 50-1000)
   */
  scroll: 150,
  /** Debounce for resize events */
  resize: 200,
  /** Debounce for hover events (when enabled) */
  hover: 200
};
var RECORDING_EVENT_CATEGORIES = {
  /**
   * CORE: Essential events for recording.
   * Always enabled - these are the fundamental user interactions.
   */
  core: {
    enabled: true,
    events: ["click", "dblclick", "contextmenu", "input", "select", "keyboard", "scroll", "navigate"]
  },
  /**
   * FOCUS: Focus and blur events.
   * Disabled by default - can be noisy and input events work without them.
   */
  focus: {
    enabled: false,
    events: ["focus", "blur"]
  },
  /**
   * HOVER: Mouse hover events.
   * Disabled by default - very noisy as users constantly move their mouse.
   * Uses debouncing when enabled.
   */
  hover: {
    enabled: false,
    events: ["hover"],
    debounceMs: 200
  },
  /**
   * DRAG_DROP: Drag and drop events.
   * Enabled by default - specialized but not noisy.
   */
  dragDrop: {
    enabled: true,
    events: ["dragstart", "drop"]
  },
  /**
   * GESTURE: Touch gesture events (swipe, tap, pinch, etc.).
   * Enabled by default - important for mobile/touch interfaces.
   */
  gesture: {
    enabled: true,
    events: ["touchstart", "touchmove", "touchend"]
  }
};
function patternsToRegExp(patterns) {
  return patterns.map((p) => new RegExp(p.pattern, p.flags));
}
function getUnstableClassPatterns() {
  return patternsToRegExp(UNSTABLE_CLASS_PATTERNS);
}
function serializeConfigForBrowser() {
  return `{
  TEST_ID_ATTRIBUTES: ${JSON.stringify(TEST_ID_ATTRIBUTES)},
  UNSTABLE_CLASS_PATTERNS: [
    ${UNSTABLE_CLASS_PATTERNS.map((p) => `new RegExp(${JSON.stringify(p.pattern)}, ${JSON.stringify(p.flags)})`).join(",\n    ")}
  ],
  DYNAMIC_ID_PATTERNS: [
    ${DYNAMIC_ID_PATTERNS.map((p) => `new RegExp(${JSON.stringify(p.pattern)}, ${JSON.stringify(p.flags)})`).join(",\n    ")}
  ],
  SEMANTIC_CLASS_PATTERNS: [
    ${SEMANTIC_CLASS_PATTERNS.map((p) => `new RegExp(${JSON.stringify(p.pattern)}, ${JSON.stringify(p.flags)})`).join(",\n    ")}
  ],
  TEXT_CONTENT_TAGS: ${JSON.stringify(TEXT_CONTENT_TAGS)},
  SELECTOR_DEFAULTS: ${JSON.stringify(SELECTOR_DEFAULTS)},
  CONFIDENCE_SCORES: ${JSON.stringify(CONFIDENCE_SCORES)},
  SPECIFICITY_SCORES: ${JSON.stringify(SPECIFICITY_SCORES)},
  RECORDING_DEBOUNCE: ${JSON.stringify(RECORDING_DEBOUNCE)},
  RECORDING_EVENT_CATEGORIES: ${JSON.stringify(RECORDING_EVENT_CATEGORIES)}
}`;
}

// src/recording/types.ts
var DEFAULT_SELECTOR_OPTIONS = {
  maxCssDepth: SELECTOR_DEFAULTS.maxCssDepth,
  includeXPath: SELECTOR_DEFAULTS.includeXPath,
  preferTestIds: SELECTOR_DEFAULTS.preferTestIds,
  unstableClassPatterns: getUnstableClassPatterns(),
  minConfidence: SELECTOR_DEFAULTS.minConfidence
};

// src/recording/validation/verification.ts
var isRecord = (value) => typeof value === "object" && value !== null && !Array.isArray(value);
var toBoolean = (value, fallback) => typeof value === "boolean" ? value : fallback;
var toNumber = (value, fallback) => typeof value === "number" && Number.isFinite(value) ? value : fallback;
var toNumberOrNull = (value) => typeof value === "number" && Number.isFinite(value) ? value : null;
var toStringOrNull = (value) => typeof value === "string" ? value : null;
var safeJsonParse = (value) => {
  try {
    return JSON.parse(value);
  } catch {
    return null;
  }
};
var parseInjectionVerification = (value) => {
  if (!isRecord(value)) {
    return null;
  }
  const loaded = value.loaded;
  if (typeof loaded !== "boolean") {
    return null;
  }
  return {
    loaded,
    loadTime: toNumberOrNull(value.loadTime),
    version: toStringOrNull(value.version),
    ready: toBoolean(value.ready, false),
    handlersCount: toNumber(value.handlersCount, 0),
    inMainContext: toBoolean(value.inMainContext, false),
    initError: toStringOrNull(value.initError) ?? void 0
  };
};
async function verifyScriptInjection(page) {
  try {
    const client = await page.context().newCDPSession(page);
    try {
      const { result } = await client.send("Runtime.evaluate", {
        expression: `(function() {
          return JSON.stringify({
            loaded: window.__vrooli_recording_script_loaded === true,
            loadTime: window.__vrooli_recording_script_load_time || null,
            version: window.__vrooli_recording_script_version || null,
            ready: window.__vrooli_recording_ready === true,
            handlersCount: window.__vrooli_recording_handlers_count || 0,
            inMainContext: window.__vrooli_recording_script_context === 'MAIN',
            initError: window.__vrooli_recording_init_error || undefined
          });
        })()`,
        returnByValue: true
      });
      if (result.type === "string" && typeof result.value === "string" && result.value) {
        const parsed = safeJsonParse(result.value);
        const verification = parseInjectionVerification(parsed);
        if (verification) {
          return verification;
        }
      }
      return {
        loaded: false,
        loadTime: null,
        version: null,
        ready: false,
        handlersCount: 0,
        inMainContext: false,
        error: "CDP evaluation returned unexpected result"
      };
    } finally {
      await client.detach().catch(() => {
      });
    }
  } catch (error) {
    return {
      loaded: false,
      loadTime: null,
      version: null,
      ready: false,
      handlersCount: 0,
      inMainContext: false,
      error: error instanceof Error ? error.message : String(error)
    };
  }
}
async function waitForScriptReady(page, timeoutMs = 5e3, pollIntervalMs = 100) {
  const startTime = Date.now();
  while (Date.now() - startTime < timeoutMs) {
    const verification = await verifyScriptInjection(page);
    if (verification.ready) {
      return verification;
    }
    if (verification.initError) {
      return verification;
    }
    await new Promise((resolve) => setTimeout(resolve, pollIntervalMs));
  }
  return verifyScriptInjection(page);
}

// src/recording/capture/init-script-generator.ts
var fs = __toESM(require("fs"));
var path = __toESM(require("path"));
var RECORDING_CONTROL_MESSAGE_TYPE = "__VROOLI_RECORDING_CONTROL__";
var DEFAULT_RECORDING_BINDING_NAME = "__vrooli_recordAction";
var cachedRecordingScriptTemplate = null;
function getRecordingScriptTemplate() {
  if (cachedRecordingScriptTemplate === null) {
    const scriptPath = path.join(__dirname, "browser-scripts", "recording-script.js");
    cachedRecordingScriptTemplate = fs.readFileSync(scriptPath, "utf-8");
  }
  return cachedRecordingScriptTemplate;
}
function generateRecordingInitScript(bindingName = DEFAULT_RECORDING_BINDING_NAME) {
  const template = getRecordingScriptTemplate();
  const serializedConfig = serializeConfigForBrowser();
  const script = template.replace("__INJECTED_CONFIG__", serializedConfig).replace(/__INJECTED_BINDING_NAME__/g, bindingName).replace(/__RECORDING_CONTROL_MESSAGE_TYPE__/g, RECORDING_CONTROL_MESSAGE_TYPE);
  return script;
}

// src/constants.ts
var MAX_SCREENSHOT_SIZE_BYTES = 512 * 1024;
var MAX_DOM_SIZE_BYTES = 512 * 1024;

// src/handlers/registry.ts
var HandlerRegistry = class {
  handlers = /* @__PURE__ */ new Map();
  /**
   * Register a handler
   *
   * Hardened: Normalizes instruction types to lowercase on registration
   * to ensure consistent lookup regardless of case in source definitions.
   */
  register(handler) {
    const types = handler.getSupportedTypes();
    for (const rawType of types) {
      const type = normalizeInstructionType(rawType);
      if (this.handlers.has(type)) {
        logger.warn("Handler already registered for type, overwriting", {
          type,
          rawType,
          handler: handler.constructor.name
        });
      }
      this.handlers.set(type, handler);
      logger.debug("Registered handler", {
        type,
        handler: handler.constructor.name
      });
    }
  }
  /**
   * Get handler for instruction type
   */
  getHandler(instruction) {
    const normalizedType = instruction.type.toLowerCase();
    const handler = this.handlers.get(normalizedType);
    if (!handler) {
      throw new UnsupportedInstructionError(instruction.type);
    }
    return handler;
  }
  /**
   * Check if type is supported
   */
  isSupported(type) {
    return this.handlers.has(type.toLowerCase());
  }
  /**
   * Get all supported types
   */
  getSupportedTypes() {
    return Array.from(this.handlers.keys());
  }
  /**
   * Get count of registered handlers
   */
  getHandlerCount() {
    return this.handlers.size;
  }
};
var handlerRegistry = new HandlerRegistry();

// src/config.ts
var import_zod2 = require("zod");
var ConfigSchema = import_zod2.z.object({
  server: import_zod2.z.object({
    port: import_zod2.z.number().min(1).max(65535).default(39400),
    host: import_zod2.z.string().default("127.0.0.1"),
    requestTimeout: import_zod2.z.number().min(1e3).max(6e5).default(3e5),
    // 5 minutes - playwright operations can be slow
    maxRequestSize: import_zod2.z.number().min(1024).max(50 * 1024 * 1024).default(5 * 1024 * 1024),
    adminSecret: import_zod2.z.string().default("")
  }),
  faultControl: import_zod2.z.object({
    // Test controls are available only in an explicitly non-production driver.
    enabled: import_zod2.z.boolean().default(false)
  }),
  browser: import_zod2.z.object({
    /**
     * Headless mode configuration.
     *
     * IMPORTANT: We use headless: false + --headless=new arg to get the "new headless" mode.
     * This makes Playwright use the regular Chromium binary (not headless_shell) with
     * the --headless=new flag, which has full service worker support.
     *
     * - headless: true  → Uses chromium_headless_shell (broken SW, causes Google loops)
     * - headless: false + --headless=new → Uses regular Chromium in new headless mode (CORRECT)
     */
    headless: import_zod2.z.boolean().default(false),
    executablePath: import_zod2.z.string().optional(),
    /**
     * Additional browser arguments.
     *
     * Default includes --headless=new for new headless mode with full SW support.
     */
    args: import_zod2.z.array(import_zod2.z.string()).default(["--headless=new"]),
    ignoreHTTPSErrors: import_zod2.z.boolean().default(false),
    /**
     * Opt-in deterministic microphone source. BrowserManager converts this to
     * Chromium's real fake-media flags; it is never enabled for normal runs.
     */
    fakeMicrophoneFile: import_zod2.z.string().min(1).optional()
  }),
  session: import_zod2.z.object({
    maxConcurrent: import_zod2.z.number().min(1).max(100).default(10),
    idleTimeoutMs: import_zod2.z.number().min(1e4).max(36e5).default(3e5),
    poolSize: import_zod2.z.number().min(1).max(50).default(5),
    cleanupIntervalMs: import_zod2.z.number().min(5e3).max(6e5).default(6e4)
  }),
  /**
   * Execution Timeouts
   *
   * These control the reliability vs speed tradeoff for instruction execution.
   * Higher values = more tolerant of slow pages, but longer waits on failures.
   * Lower values = faster failure detection, but may fail on slow networks/pages.
   *
   * Recommended tuning:
   * - Fast local testing: reduce timeouts by 50%
   * - Flaky/slow sites: increase timeouts by 50-100%
   * - CI environments: use defaults or slightly higher
   */
  execution: import_zod2.z.object({
    /** Default timeout for general operations (click, type, etc.) - ms */
    defaultTimeoutMs: import_zod2.z.number().min(1e3).max(3e5).default(3e4),
    /** Navigation timeout (goto, reload) - typically longer due to network - ms */
    navigationTimeoutMs: import_zod2.z.number().min(5e3).max(3e5).default(45e3),
    /** Wait timeout (waitForSelector, waitForTimeout) - ms */
    waitTimeoutMs: import_zod2.z.number().min(1e3).max(3e5).default(3e4),
    /** Assertion timeout - typically shorter for faster feedback - ms */
    assertionTimeoutMs: import_zod2.z.number().min(1e3).max(12e4).default(15e3),
    /** Replay action timeout during recording preview - ms */
    replayActionTimeoutMs: import_zod2.z.number().min(1e3).max(12e4).default(1e4)
  }),
  /**
   * Recording Configuration
   *
   * Controls how Record Mode captures and processes user actions.
   *
   * Buffer size tradeoff: larger = more actions stored, higher memory usage.
   * Selector confidence: higher = stricter selector selection, fewer candidates.
   */
  recording: import_zod2.z.object({
    /** Maximum actions to buffer in memory per session (FIFO eviction when full) */
    maxBufferSize: import_zod2.z.number().min(100).max(1e5).default(1e4),
    /** Minimum confidence score (0-1) for selector candidates to be included */
    minSelectorConfidence: import_zod2.z.number().min(0).max(1).default(0.3),
    /** Default swipe gesture distance in pixels */
    defaultSwipeDistance: import_zod2.z.number().min(50).max(1e3).default(300),
    /**
     * Enable verbose recording diagnostics.
     * When true, logs detailed information about:
     * - Script injection (success/failure, injection method)
     * - Event flow (events received, parsed, converted)
     * - Context isolation debugging
     *
     * Controlled by RECORDING_DIAGNOSTICS_ENABLED environment variable.
     */
    diagnosticsEnabled: import_zod2.z.boolean().default(false),
    /**
     * Debounce timings for event capture (ms).
     * Lower = more responsive, more events. Higher = more batching, fewer events.
     */
    debounce: import_zod2.z.object({
      /** Input event debounce - batches keystrokes into single type actions */
      inputMs: import_zod2.z.number().min(50).max(2e3).default(500),
      /** Scroll event debounce - reduces scroll event noise */
      scrollMs: import_zod2.z.number().min(50).max(1e3).default(150)
    }),
    /**
     * Selector Generation Options
     * Controls how selectors are generated for recorded elements.
     */
    selector: import_zod2.z.object({
      /** Maximum CSS path depth for traversal (lower = shorter selectors, may be less unique) */
      maxCssDepth: import_zod2.z.number().min(2).max(10).default(5),
      /** Whether to include XPath as a fallback selector strategy */
      includeXPath: import_zod2.z.boolean().default(true)
    })
  }),
  telemetry: import_zod2.z.object({
    screenshot: import_zod2.z.object({
      enabled: import_zod2.z.boolean().default(true),
      fullPage: import_zod2.z.boolean().default(false),
      // Default false to prevent viewport oscillation during execution
      quality: import_zod2.z.number().min(1).max(100).default(80),
      maxSizeBytes: import_zod2.z.number().min(1024).max(10 * 1024 * 1024).default(512e3)
    }),
    dom: import_zod2.z.object({
      enabled: import_zod2.z.boolean().default(true),
      maxSizeBytes: import_zod2.z.number().min(1024).max(10 * 1024 * 1024).default(524288)
    }),
    console: import_zod2.z.object({
      enabled: import_zod2.z.boolean().default(true),
      maxEntries: import_zod2.z.number().min(1).max(1e4).default(100)
    }),
    network: import_zod2.z.object({
      enabled: import_zod2.z.boolean().default(true),
      maxEvents: import_zod2.z.number().min(1).max(1e4).default(200)
    }),
    har: import_zod2.z.object({
      enabled: import_zod2.z.boolean().default(false)
    }),
    tracing: import_zod2.z.object({
      enabled: import_zod2.z.boolean().default(false)
    })
  }),
  logging: import_zod2.z.object({
    level: import_zod2.z.enum(["debug", "info", "warn", "error"]).default("info"),
    format: import_zod2.z.enum(["json", "text"]).default("json")
  }),
  metrics: import_zod2.z.object({
    enabled: import_zod2.z.boolean().default(true),
    port: import_zod2.z.number().min(1).max(65535).default(9090)
  }),
  /**
   * Frame Streaming Configuration
   *
   * Controls how frames are captured and streamed during recording.
   * The default uses CDP startScreencast for push-based frame delivery,
   * with fallback to polling-based screenshot capture.
   *
   * CDP screencast provides 30-60 FPS vs 10-15 FPS with polling.
   *
   * CONTROL LEVERS:
   * - useScreencast: Strategy selection (CDP push vs polling pull)
   * - fallbackToPolling: Resilience strategy (auto-fallback on failure)
   * - cdp.ackTimeoutMs: CDP responsiveness vs tolerance tradeoff
   * - cdp.maxAckFailures: Failure tolerance before logging errors
   * - cdp.frameLogInterval: Observability vs log noise tradeoff
   * - cdp.pageCheckIntervalMs: Multi-tab responsiveness vs CPU tradeoff
   */
  frameStreaming: import_zod2.z.object({
    /** Use CDP screencast (true) or legacy polling (false) */
    useScreencast: import_zod2.z.boolean().default(true),
    /** Fall back to polling if screencast fails */
    fallbackToPolling: import_zod2.z.boolean().default(true),
    /**
     * CDP-specific tuning options.
     * These control the behavior of the CDP screencast strategy.
     */
    cdp: import_zod2.z.object({
      /**
       * Timeout (ms) for acknowledging each screencast frame.
       * Chrome stops sending frames if ACKs are not received.
       * Trade-off: Lower = faster failure detection, but may cause spurious timeouts on slow systems.
       * Higher = more tolerant of system load spikes, but delays error detection.
       */
      ackTimeoutMs: import_zod2.z.number().min(100).max(1e4).default(1e3),
      /**
       * Maximum consecutive ACK failures before logging an error.
       * Trade-off: Lower = earlier warning, more noise. Higher = quieter logs, later detection.
       */
      maxAckFailures: import_zod2.z.number().min(1).max(100).default(5),
      /**
       * Log frame statistics every N frames.
       * Trade-off: Lower = more visibility, more log volume. Higher = less noise, less observability.
       * Set to 0 to disable periodic logging.
       */
      frameLogInterval: import_zod2.z.number().min(0).max(1e3).default(60),
      /**
       * Interval (ms) for checking if the active page has changed (multi-tab support).
       * Screencast must be restarted when tabs change.
       * Trade-off: Lower = faster tab switch detection, more CPU. Higher = less overhead, slower detection.
       */
      pageCheckIntervalMs: import_zod2.z.number().min(50).max(5e3).default(100)
    })
  }),
  /**
   * Performance Debug Mode
   *
   * Controls timing instrumentation for the frame streaming pipeline.
   * When enabled, detailed timing data is collected and can be sent
   * to the API for aggregation and analysis.
   *
   * Tradeoff: Enabling adds ~1-2ms overhead per frame for timing collection.
   */
  performance: import_zod2.z.object({
    /** Enable debug performance mode for frame streaming */
    enabled: import_zod2.z.boolean().default(false),
    /** Include timing data in WebSocket frame headers (prepended to JPEG) */
    includeTimingHeaders: import_zod2.z.boolean().default(true),
    /** Log performance summaries every N frames (0 = disabled) */
    logSummaryInterval: import_zod2.z.number().min(0).max(1e3).default(60),
    /** Number of frame timings to retain for percentile analysis */
    bufferSize: import_zod2.z.number().min(1).max(1e3).default(100)
  }),
  /**
   * History Callback Configuration
   *
   * Controls how navigation history is reported to the Go API for persistence.
   * When a callback URL is configured, navigation events are POSTed to that URL
   * with page title, URL, timestamp, and optional thumbnail.
   */
  history: import_zod2.z.object({
    /** URL to POST history entries when navigation occurs (empty = disabled) */
    callbackUrl: import_zod2.z.string().default(""),
    /** Whether to capture thumbnails for history entries */
    thumbnailEnabled: import_zod2.z.boolean().default(true),
    /** JPEG quality for history thumbnails (10-100) */
    thumbnailQuality: import_zod2.z.number().min(10).max(100).default(60)
  })
});
function parseRequiredEnvInt(envName) {
  const envVar = process.env[envName];
  if (!envVar || envVar.trim() === "") {
    throw new Error(`Required environment variable ${envName} is not set. This should be set by the vrooli lifecycle system.`);
  }
  const parsed = parseInt(envVar, 10);
  if (Number.isNaN(parsed)) {
    throw new Error(`Environment variable ${envName} has invalid value "${envVar}" - expected an integer.`);
  }
  return parsed;
}
function parseEnvInt(envVar, defaultValue) {
  if (!envVar || envVar.trim() === "") {
    return defaultValue;
  }
  const parsed = parseInt(envVar, 10);
  if (Number.isNaN(parsed)) {
    console.warn(`Invalid numeric config value: "${envVar}", using default: ${defaultValue}`);
    return defaultValue;
  }
  return parsed;
}
function parseLogLevel(envVar) {
  const validLevels = ["debug", "info", "warn", "error"];
  const level = (envVar || "info").toLowerCase();
  if (!validLevels.includes(level)) {
    console.warn(`Invalid LOG_LEVEL "${envVar}", using default: info`);
    return "info";
  }
  return level;
}
function parseLogFormat(envVar) {
  const validFormats = ["json", "text"];
  const format = (envVar || "json").toLowerCase();
  if (!validFormats.includes(format)) {
    console.warn(`Invalid LOG_FORMAT "${envVar}", using default: json`);
    return "json";
  }
  return format;
}
function parseEnvFloat(envVar, defaultValue) {
  if (!envVar || envVar.trim() === "") {
    return defaultValue;
  }
  const parsed = parseFloat(envVar);
  if (Number.isNaN(parsed)) {
    console.warn(`Invalid numeric config value: "${envVar}", using default: ${defaultValue}`);
    return defaultValue;
  }
  return parsed;
}
function loadConfig() {
  const config = {
    server: {
      port: parseRequiredEnvInt("PLAYWRIGHT_DRIVER_PORT"),
      host: process.env.PLAYWRIGHT_DRIVER_HOST || "127.0.0.1",
      requestTimeout: parseEnvInt(process.env.REQUEST_TIMEOUT_MS, 3e5),
      // 5 minutes
      maxRequestSize: parseEnvInt(process.env.MAX_REQUEST_SIZE, 5242880),
      adminSecret: process.env.PLAYWRIGHT_DRIVER_ADMIN_SECRET?.trim() || ""
    },
    faultControl: {
      enabled: process.env.NODE_ENV !== "production" && process.env.PLAYWRIGHT_DRIVER_FAULT_CONTROL !== "false"
    },
    browser: {
      // Default to false to use regular Chromium with --headless=new (not headless_shell)
      headless: process.env.HEADLESS === "true",
      executablePath: process.env.BROWSER_EXECUTABLE_PATH,
      // Hardened: Simple comma-split can break args containing commas (rare but possible)
      // Browser args that contain commas are uncommon, but for robustness we trim each arg
      // and filter out empty strings that might result from trailing/leading commas
      // Note: If you need args with commas, use a different delimiter in the env var
      // e.g., BROWSER_ARGS="--arg1;;--arg2=value" with split(';;')
      //
      // Default includes:
      // - --headless=new: Uses new headless mode with full service worker support
      // - --disable-blink-features=AutomationControlled: Hides automation detection signals
      // These fix Google redirect loop issues caused by bot detection
      args: [
        "--headless=new",
        "--disable-blink-features=AutomationControlled",
        ...process.env.BROWSER_ARGS ? process.env.BROWSER_ARGS.split(",").map((arg) => arg.trim()).filter((arg) => arg.length > 0) : []
      ],
      ignoreHTTPSErrors: process.env.IGNORE_HTTPS_ERRORS === "true",
      fakeMicrophoneFile: process.env.BAS_FAKE_MICROPHONE_FILE?.trim() || void 0
    },
    session: {
      maxConcurrent: parseEnvInt(process.env.MAX_SESSIONS, 10),
      idleTimeoutMs: parseEnvInt(process.env.SESSION_IDLE_TIMEOUT_MS, 3e5),
      poolSize: parseEnvInt(process.env.SESSION_POOL_SIZE, 5),
      cleanupIntervalMs: parseEnvInt(process.env.CLEANUP_INTERVAL_MS, 6e4)
    },
    // Execution timeouts - the main performance/reliability tradeoff
    execution: {
      defaultTimeoutMs: parseEnvInt(process.env.EXECUTION_DEFAULT_TIMEOUT_MS, 3e4),
      navigationTimeoutMs: parseEnvInt(process.env.EXECUTION_NAVIGATION_TIMEOUT_MS, 45e3),
      waitTimeoutMs: parseEnvInt(process.env.EXECUTION_WAIT_TIMEOUT_MS, 3e4),
      assertionTimeoutMs: parseEnvInt(process.env.EXECUTION_ASSERTION_TIMEOUT_MS, 15e3),
      replayActionTimeoutMs: parseEnvInt(process.env.EXECUTION_REPLAY_TIMEOUT_MS, 1e4)
    },
    // Recording configuration
    recording: {
      maxBufferSize: parseEnvInt(process.env.RECORDING_MAX_BUFFER_SIZE, 1e4),
      minSelectorConfidence: parseEnvFloat(process.env.RECORDING_MIN_SELECTOR_CONFIDENCE, 0.3),
      defaultSwipeDistance: parseEnvInt(process.env.RECORDING_DEFAULT_SWIPE_DISTANCE, 300),
      diagnosticsEnabled: process.env.RECORDING_DIAGNOSTICS_ENABLED === "true",
      debounce: {
        inputMs: parseEnvInt(process.env.RECORDING_INPUT_DEBOUNCE_MS, 500),
        scrollMs: parseEnvInt(process.env.RECORDING_SCROLL_DEBOUNCE_MS, 150)
      },
      selector: {
        maxCssDepth: parseEnvInt(process.env.RECORDING_MAX_CSS_DEPTH, 5),
        includeXPath: process.env.RECORDING_INCLUDE_XPATH !== "false"
      }
    },
    telemetry: {
      screenshot: {
        enabled: process.env.SCREENSHOT_ENABLED !== "false",
        // Default to false to prevent viewport oscillation during execution
        // Set SCREENSHOT_FULL_PAGE=true to enable full page screenshots
        fullPage: process.env.SCREENSHOT_FULL_PAGE === "true",
        quality: parseEnvInt(process.env.SCREENSHOT_QUALITY, 80),
        maxSizeBytes: parseEnvInt(process.env.SCREENSHOT_MAX_SIZE, 512e3)
      },
      dom: {
        enabled: process.env.DOM_ENABLED !== "false",
        maxSizeBytes: parseEnvInt(process.env.DOM_MAX_SIZE, 524288)
      },
      console: {
        enabled: process.env.CONSOLE_ENABLED !== "false",
        maxEntries: parseEnvInt(process.env.CONSOLE_MAX_ENTRIES, 100)
      },
      network: {
        enabled: process.env.NETWORK_ENABLED !== "false",
        maxEvents: parseEnvInt(process.env.NETWORK_MAX_EVENTS, 200)
      },
      har: {
        enabled: process.env.HAR_ENABLED === "true"
      },
      tracing: {
        enabled: process.env.TRACING_ENABLED === "true"
      }
    },
    logging: {
      level: parseLogLevel(process.env.LOG_LEVEL),
      format: parseLogFormat(process.env.LOG_FORMAT)
    },
    metrics: {
      enabled: process.env.METRICS_ENABLED !== "false",
      port: parseEnvInt(process.env.METRICS_PORT, 9090)
    },
    frameStreaming: {
      useScreencast: process.env.FRAME_STREAMING_USE_SCREENCAST !== "false",
      // Default true
      fallbackToPolling: process.env.FRAME_STREAMING_FALLBACK !== "false",
      // Default true
      cdp: {
        ackTimeoutMs: parseEnvInt(process.env.FRAME_STREAMING_CDP_ACK_TIMEOUT_MS, 1e3),
        maxAckFailures: parseEnvInt(process.env.FRAME_STREAMING_CDP_MAX_ACK_FAILURES, 5),
        frameLogInterval: parseEnvInt(process.env.FRAME_STREAMING_CDP_FRAME_LOG_INTERVAL, 60),
        pageCheckIntervalMs: parseEnvInt(process.env.FRAME_STREAMING_CDP_PAGE_CHECK_INTERVAL_MS, 100)
      }
    },
    performance: {
      enabled: process.env.PLAYWRIGHT_DRIVER_PERF_ENABLED === "true",
      includeTimingHeaders: process.env.PLAYWRIGHT_DRIVER_PERF_INCLUDE_HEADERS !== "false",
      logSummaryInterval: parseEnvInt(process.env.PLAYWRIGHT_DRIVER_PERF_LOG_INTERVAL, 60),
      bufferSize: parseEnvInt(process.env.PLAYWRIGHT_DRIVER_PERF_BUFFER_SIZE, 100)
    },
    history: {
      callbackUrl: process.env.HISTORY_CALLBACK_URL || "",
      thumbnailEnabled: process.env.HISTORY_THUMBNAIL_ENABLED !== "false",
      thumbnailQuality: parseEnvInt(process.env.HISTORY_THUMBNAIL_QUALITY, 60)
    }
  };
  const parsed = ConfigSchema.parse(config);
  return deepFreeze(parsed);
}
function deepFreeze(obj) {
  const propNames = Object.getOwnPropertyNames(obj);
  for (const name of propNames) {
    const value = obj[name];
    if (value && typeof value === "object" && !Object.isFrozen(value)) {
      deepFreeze(value);
    }
  }
  return Object.freeze(obj);
}

// src/recording/handler-adapter.ts
function timelineEntryToHandlerInstruction(entry) {
  const actionType = entry.action?.type ?? 0 /* UNSPECIFIED */;
  return {
    index: entry.sequenceNum,
    nodeId: entry.id,
    type: actionTypeToString(actionType),
    params: {},
    // Legacy field, handlers use action.params
    action: entry.action
  };
}
var cachedConfig;
var cachedLogger;
var cachedMetrics;
function createReplayHandlerContext(replayContext, sessionId = "replay-session") {
  if (!cachedConfig) {
    cachedConfig = loadConfig();
    cachedConfig.execution.defaultTimeoutMs = replayContext.timeout;
    cachedConfig.execution.navigationTimeoutMs = replayContext.timeout;
    cachedConfig.execution.waitTimeoutMs = replayContext.timeout;
  }
  if (!cachedLogger) {
    cachedLogger = createNoOpLogger();
  }
  if (!cachedMetrics) {
    cachedMetrics = createNoOpMetrics();
  }
  const browserContext = replayContext.page.context();
  return {
    page: replayContext.page,
    browserContext,
    config: cachedConfig,
    logger: cachedLogger,
    // Cast to Metrics - NoOpMetrics implements the same interface
    metrics: cachedMetrics,
    sessionId
  };
}
async function executeViaHandler(entry, replayContext, sessionId) {
  const startTime = Date.now();
  const instruction = timelineEntryToHandlerInstruction(entry);
  try {
    if (!handlerRegistry.isSupported(instruction.type)) {
      return {
        success: false,
        durationMs: Date.now() - startTime,
        error: {
          message: `No handler registered for action type: ${instruction.type}`,
          code: "UNSUPPORTED_ACTION"
        }
      };
    }
    const handler = handlerRegistry.getHandler(instruction);
    const handlerContext = createReplayHandlerContext(replayContext, sessionId);
    const result = await handler.execute(instruction, handlerContext);
    return {
      success: result.success,
      durationMs: Date.now() - startTime,
      error: result.error ? {
        message: result.error.message,
        code: result.error.code || "UNKNOWN"
      } : void 0
    };
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    let code = "UNKNOWN";
    if (message.includes("waiting for selector") || message.includes("Timeout")) {
      code = "TIMEOUT";
    } else if (message.includes("not visible")) {
      code = "ELEMENT_NOT_VISIBLE";
    } else if (message.includes("not enabled") || message.includes("disabled")) {
      code = "ELEMENT_NOT_ENABLED";
    } else if (message.includes("UnsupportedInstructionError")) {
      code = "UNSUPPORTED_ACTION";
    }
    return {
      success: false,
      durationMs: Date.now() - startTime,
      error: { message, code }
    };
  }
}
function hasHandlerForActionType(actionType) {
  const typeString = actionTypeToString(actionType);
  return handlerRegistry.isSupported(typeString);
}

// src/recording/action-executor.ts
var executorRegistry = /* @__PURE__ */ new Map();
function registerTimelineExecutor(actionType, executor) {
  if (executorRegistry.has(actionType)) {
    logger.warn(scopedLog(LogContext.RECORDING, "overwriting executor for action type"), {
      actionType: ActionType[actionType]
    });
  }
  executorRegistry.set(actionType, executor);
}
function createBaseResult(entry) {
  return {
    entryId: entry.id,
    sequenceNum: entry.sequenceNum,
    actionType: entry.action?.type ?? 0 /* UNSPECIFIED */,
    success: false,
    durationMs: 0
  };
}
function unsupportedResult(base) {
  return {
    ...base,
    error: {
      message: `Unsupported action type for replay: ${ActionType[base.actionType]}`,
      code: "UNSUPPORTED_ACTION"
    }
  };
}
function createHandlerDelegatedExecutor(actionType) {
  return async (entry, context) => {
    const base = createBaseResult(entry);
    if (!hasHandlerForActionType(actionType)) {
      return unsupportedResult(base);
    }
    const replayContext = {
      page: context.page,
      timeout: context.timeout,
      validateSelector: context.validateSelector
    };
    const result = await executeViaHandler(entry, replayContext);
    return {
      ...base,
      success: result.success,
      durationMs: result.durationMs,
      error: result.error ? {
        message: result.error.message,
        code: result.error.code,
        matchCount: result.error.matchCount,
        selector: result.error.selector
      } : void 0
    };
  };
}
var allHandlerDelegatedActions = [
  // Core actions (previously had inline executors)
  1 /* NAVIGATE */,
  2 /* CLICK */,
  3 /* INPUT */,
  6 /* SCROLL */,
  7 /* SELECT */,
  9 /* KEYBOARD */,
  12 /* FOCUS */,
  10 /* HOVER */,
  13 /* BLUR */,
  4 /* WAIT */,
  11 /* SCREENSHOT */,
  5 /* ASSERT */,
  8 /* EVALUATE */,
  // Extended actions (always used handler delegation)
  14 /* SUBFLOW */,
  15 /* EXTRACT */,
  16 /* UPLOAD_FILE */,
  17 /* DOWNLOAD */,
  18 /* FRAME_SWITCH */,
  19 /* TAB_SWITCH */,
  20 /* COOKIE_STORAGE */,
  21 /* SHORTCUT */,
  22 /* DRAG_DROP */,
  23 /* GESTURE */,
  24 /* NETWORK_MOCK */,
  25 /* ROTATE */
];
for (const actionType of allHandlerDelegatedActions) {
  registerTimelineExecutor(actionType, createHandlerDelegatedExecutor(actionType));
}

// src/infra/session-cleanup-registry.ts
var registrations = [];
function registerSessionCleanup(name, cleanup) {
  const existing = registrations.find((r) => r.name === name);
  if (existing) {
    logger.warn(scopedLog(LogContext.CLEANUP, "duplicate cleanup registration"), {
      name,
      hint: "Cleanup function with this name already registered. Ignoring duplicate."
    });
    return;
  }
  registrations.push({ name, cleanup });
  logger.debug(scopedLog(LogContext.CLEANUP, "cleanup function registered"), {
    name,
    totalRegistrations: registrations.length
  });
}

// src/infra/operation-tracker.ts
var DEFAULT_CONFIG = {
  resultCacheTtlMs: 3e5,
  // 5 minutes
  cacheResults: false,
  maxCachedPerSession: 100
};
function createOperationTracker(config) {
  const cfg = { ...DEFAULT_CONFIG, ...config };
  const inFlight = /* @__PURE__ */ new Map();
  const cached = /* @__PURE__ */ new Map();
  function getSessionInFlight(sessionId) {
    let session = inFlight.get(sessionId);
    if (!session) {
      session = /* @__PURE__ */ new Map();
      inFlight.set(sessionId, session);
    }
    return session;
  }
  function getSessionCached(sessionId) {
    let session = cached.get(sessionId);
    if (!session) {
      session = /* @__PURE__ */ new Map();
      cached.set(sessionId, session);
    }
    return session;
  }
  const tracker = {
    getInFlight(sessionId, operationKey) {
      const session = inFlight.get(sessionId);
      if (!session) return null;
      const pending = session.get(operationKey);
      if (pending) {
        logger.debug(scopedLog(LogContext.INSTRUCTION, `${cfg.name}: awaiting in-flight operation`), {
          sessionId,
          operationKey
        });
        return pending;
      }
      return null;
    },
    trackInFlight(sessionId, operationKey, promise) {
      const session = getSessionInFlight(sessionId);
      session.set(operationKey, promise);
      logger.debug(scopedLog(LogContext.INSTRUCTION, `${cfg.name}: tracking in-flight operation`), {
        sessionId,
        operationKey
      });
      return () => {
        session.delete(operationKey);
        if (session.size === 0) {
          inFlight.delete(sessionId);
        }
      };
    },
    getCached(sessionId, operationKey) {
      if (!cfg.cacheResults) return null;
      const session = cached.get(sessionId);
      if (!session) return null;
      const entry = session.get(operationKey);
      if (!entry) return null;
      if (Date.now() - entry.timestamp > cfg.resultCacheTtlMs) {
        session.delete(operationKey);
        return null;
      }
      logger.debug(scopedLog(LogContext.INSTRUCTION, `${cfg.name}: returning cached result`), {
        sessionId,
        operationKey,
        cacheAgeMs: Date.now() - entry.timestamp
      });
      return entry.result;
    },
    cacheResult(sessionId, operationKey, result) {
      if (!cfg.cacheResults) return;
      const session = getSessionCached(sessionId);
      if (session.size >= cfg.maxCachedPerSession) {
        const firstKey = session.keys().next().value;
        if (firstKey) {
          session.delete(firstKey);
        }
      }
      session.set(operationKey, {
        result,
        timestamp: Date.now()
      });
      logger.debug(scopedLog(LogContext.INSTRUCTION, `${cfg.name}: cached result`), {
        sessionId,
        operationKey,
        cacheSize: session.size
      });
    },
    clearSession(sessionId) {
      const inFlightCount = inFlight.get(sessionId)?.size ?? 0;
      const cachedCount = cached.get(sessionId)?.size ?? 0;
      inFlight.delete(sessionId);
      cached.delete(sessionId);
      if (inFlightCount > 0 || cachedCount > 0) {
        logger.debug(scopedLog(LogContext.CLEANUP, `${cfg.name}: cleared session state`), {
          sessionId,
          inFlightCleared: inFlightCount,
          cachedCleared: cachedCount
        });
      }
    },
    getName() {
      return cfg.name;
    }
  };
  registerSessionCleanup(`operation-tracker:${cfg.name}`, (sessionId) => {
    tracker.clearSession(sessionId);
  });
  return tracker;
}
var downloadTracker = createOperationTracker({
  name: "download",
  cacheResults: true,
  resultCacheTtlMs: 3e5
  // 5 minutes
});
var uploadTracker = createOperationTracker({
  name: "upload",
  cacheResults: false
  // Uploads don't need result caching
});
var tabTracker = createOperationTracker({
  name: "tab",
  cacheResults: false
});

// src/infra/in-flight-guard.ts
function createSetGuard(_config) {
  const set = /* @__PURE__ */ new Set();
  return {
    has: (key) => set.has(key),
    add: (key) => {
      set.add(key);
    },
    delete: (key) => set.delete(key),
    size: () => set.size,
    clear: () => set.clear()
  };
}
function createWeakSetGuard() {
  const set = /* @__PURE__ */ new WeakSet();
  return {
    has: (key) => set.has(key),
    add: (key) => {
      set.add(key);
    },
    delete: (key) => set.delete(key)
  };
}

// src/playwright/provider.ts
var import_rebrowser_playwright = require("rebrowser-playwright");
var DEFAULT_PROVIDER = "rebrowser-playwright";
function getConfiguredProviderName() {
  const envProvider = process.env.PLAYWRIGHT_PROVIDER;
  if (envProvider === "playwright" || envProvider === "rebrowser-playwright") {
    return envProvider;
  }
  return DEFAULT_PROVIDER;
}
var PROVIDER_CAPABILITIES = {
  "rebrowser-playwright": {
    /**
     * rebrowser-playwright runs page.evaluate() in ISOLATED context
     * to prevent bot detection. This means:
     * - Scripts can't wrap History API in the page
     * - Scripts can't access page's global variables
     * - Scripts are invisible to page's JavaScript
     *
     * Workaround: Inject scripts via HTML modification (route interception)
     */
    evaluateIsolated: true,
    /**
     * exposeBinding() callbacks only fire when called from ISOLATED context
     * (i.e., from page.evaluate()). Scripts in MAIN context (injected HTML)
     * cannot trigger bindings directly.
     *
     * Workaround: Use fetch() to a route-intercepted URL for event communication
     */
    exposeBindingIsolated: true,
    /**
     * rebrowser-playwright includes patches that:
     * - Hide navigator.webdriver
     * - Mask automation markers
     * - Pass common bot detection checks
     *
     * This is essential for recording on production websites that
     * block detected automation.
     */
    hasAntiDetection: true
  },
  playwright: {
    /**
     * Standard playwright runs page.evaluate() in MAIN context.
     * Scripts can wrap History API and access page globals.
     * But they are visible to page's JavaScript (detectable).
     */
    evaluateIsolated: false,
    /**
     * Standard playwright's exposeBinding() works from any context.
     * Scripts injected via addInitScript can call bindings directly.
     */
    exposeBindingIsolated: false,
    /**
     * Standard playwright has no anti-detection.
     * navigator.webdriver = true, easily detected.
     */
    hasAntiDetection: false
  }
};
function createPlaywrightProvider(name) {
  const providerName = name ?? getConfiguredProviderName();
  const capabilities = PROVIDER_CAPABILITIES[providerName];
  if (providerName === "playwright") {
    logger.warn(
      "[playwright-provider] Standard playwright requested but only rebrowser-playwright is installed. To use standard playwright: pnpm add playwright. Using rebrowser-playwright instead."
    );
    return {
      name: "rebrowser-playwright",
      chromium: import_rebrowser_playwright.chromium,
      capabilities: PROVIDER_CAPABILITIES["rebrowser-playwright"]
    };
  }
  return {
    name: providerName,
    chromium: import_rebrowser_playwright.chromium,
    capabilities
  };
}
var playwrightProvider = createPlaywrightProvider();

// src/recording/orchestration/decisions.ts
function shouldInjectScript(resourceType, url, status, contentType) {
  if (resourceType !== "document") {
    return {
      shouldInject: false,
      reason: "skip_non_document",
      message: `Skipping non-document request (resourceType=${resourceType})`,
      metadata: { resourceType, url: url.slice(0, 80) }
    };
  }
  if (url.includes("__vrooli_recording_test__")) {
    return {
      shouldInject: false,
      reason: "skip_test_page",
      message: "Skipping test page URL (handled by dedicated route)",
      metadata: { url: url.slice(0, 80) }
    };
  }
  if (status !== void 0 && status >= 300 && status < 400) {
    return {
      shouldInject: false,
      reason: "skip_redirect",
      message: `Skipping redirect response (status=${status})`,
      metadata: { status, url: url.slice(0, 80) }
    };
  }
  if (contentType !== void 0 && !contentType.includes("text/html")) {
    return {
      shouldInject: false,
      reason: "skip_non_html",
      message: `Skipping non-HTML response (contentType=${contentType.slice(0, 50)})`,
      metadata: { contentType: contentType.slice(0, 50), url: url.slice(0, 80) }
    };
  }
  return {
    shouldInject: true,
    reason: "yes_html_document",
    message: "HTML document - injecting recording script",
    metadata: { resourceType, url: url.slice(0, 80), status, contentType: contentType?.slice(0, 50) }
  };
}
function shouldProcessEvent(phase, hasHandler, eventType, generation, eventGeneration) {
  if (phase !== "capturing") {
    return {
      shouldProcess: false,
      reason: "not_recording",
      message: `Event dropped: not in capturing phase (phase=${phase})`,
      metadata: { phase, eventType }
    };
  }
  if (!hasHandler) {
    return {
      shouldProcess: false,
      reason: "no_handler",
      message: `Event dropped: no event handler set`,
      metadata: { phase, hasHandler: false, eventType }
    };
  }
  if (generation !== void 0 && eventGeneration !== void 0 && eventGeneration !== generation) {
    return {
      shouldProcess: false,
      reason: "stale_generation",
      message: `Event dropped: stale generation (event=${eventGeneration}, current=${generation})`,
      metadata: { phase, eventType }
    };
  }
  return {
    shouldProcess: true,
    reason: "accepted",
    message: `Event accepted: ${eventType}`,
    metadata: { phase, hasHandler: true, eventType }
  };
}
function formatDecisionForLog(decision) {
  if ("shouldInject" in decision) {
    return {
      decision: decision.shouldInject ? "INJECT" : "SKIP",
      reason: decision.reason,
      message: decision.message,
      ...decision.metadata
    };
  }
  if ("shouldProcess" in decision) {
    return {
      decision: decision.shouldProcess ? "PROCESS" : "DROP",
      reason: decision.reason,
      message: decision.message,
      ...decision.metadata
    };
  }
  return {
    decision: decision.canStart ? "READY" : "BLOCKED",
    blockers: decision.blockers,
    suggestions: decision.suggestions,
    ...decision.metadata
  };
}

// src/recording/io/event-route.ts
var RECORDING_EVENT_URL = "/__vrooli_recording_event__";
function createRouteHandlerStats() {
  return {
    eventsReceived: 0,
    eventsProcessed: 0,
    eventsDroppedNoHandler: 0,
    eventsWithErrors: 0,
    lastEventAt: null,
    lastEventType: null
  };
}
function cloneRouteHandlerStats(stats) {
  return { ...stats };
}
function resetRouteHandlerStats(stats) {
  stats.eventsReceived = 0;
  stats.eventsProcessed = 0;
  stats.eventsDroppedNoHandler = 0;
  stats.eventsWithErrors = 0;
  stats.lastEventAt = null;
  stats.lastEventType = null;
}
function createEventRouteManager(options) {
  const { logger: logger2, getEventHandler } = options;
  const stats = createRouteHandlerStats();
  const pagesWithEventRoute = createWeakSetGuard();
  const pagesBeingSetUp = createSetGuard({ name: "pages-setup-lock" });
  function handleRecordingEvent(postData) {
    const rawEvent = JSON.parse(postData);
    stats.lastEventType = rawEvent.actionType;
    const eventHandler = getEventHandler();
    const decision = shouldProcessEvent(
      eventHandler ? "capturing" : "idle",
      !!eventHandler,
      rawEvent.actionType
    );
    logger2.info(
      scopedLog(LogContext.RECORDING, `event decision: ${decision.reason}`),
      formatDecisionForLog(decision)
    );
    if (decision.shouldProcess && eventHandler) {
      try {
        eventHandler(rawEvent);
        stats.eventsProcessed++;
      } catch (error) {
        stats.eventsWithErrors++;
        logger2.error(scopedLog(LogContext.RECORDING, "event handler error"), {
          error: error instanceof Error ? error.message : String(error),
          actionType: rawEvent.actionType
        });
      }
    } else {
      stats.eventsDroppedNoHandler++;
      logger2.warn(
        scopedLog(LogContext.RECORDING, `event dropped: ${decision.reason}`),
        formatDecisionForLog(decision)
      );
    }
  }
  async function setupPageEventRoute(page, routeOptions = {}) {
    const { force = false } = routeOptions;
    if (!force && pagesWithEventRoute.has(page)) {
      logger2.debug(scopedLog(LogContext.RECORDING, "page event route already set up, skipping"), {
        url: page.url()?.slice(0, 50)
      });
      return;
    }
    if (!force && pagesBeingSetUp.has(page)) {
      logger2.debug(scopedLog(LogContext.RECORDING, "page event route setup already in progress, skipping"), {
        url: page.url()?.slice(0, 50)
      });
      return;
    }
    pagesBeingSetUp.add(page);
    try {
      await page.route(`**${RECORDING_EVENT_URL}`, async (route) => {
        try {
          const request = route.request();
          stats.eventsReceived++;
          stats.lastEventAt = (/* @__PURE__ */ new Date()).toISOString();
          logger2.info(scopedLog(LogContext.RECORDING, "page-level event route matched"), {
            url: request.url().slice(0, 100),
            method: request.method(),
            eventsReceived: stats.eventsReceived
          });
          const postData = request.postData();
          if (postData) {
            handleRecordingEvent(postData);
          }
          await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: '{"ok":true}'
          });
        } catch (error) {
          logger2.error(scopedLog(LogContext.RECORDING, "page event route handler error"), {
            error: error instanceof Error ? error.message : String(error)
          });
          await route.fulfill({ status: 500, body: "error" });
        }
      });
      pagesWithEventRoute.add(page);
      logger2.info(scopedLog(LogContext.RECORDING, "page-level event route set up"), {
        url: page.url()?.slice(0, 50)
      });
    } finally {
      pagesBeingSetUp.delete(page);
    }
  }
  return {
    setupPageEventRoute,
    getStats: () => cloneRouteHandlerStats(stats),
    resetStats: () => resetRouteHandlerStats(stats),
    hasEventRoute: (page) => pagesWithEventRoute.has(page)
  };
}

// src/recording/injection/types.ts
function createInitialStats() {
  return {
    attempted: 0,
    successful: 0,
    failed: 0,
    avgInjectionTimeMs: 0,
    lastInjectionAt: null
  };
}
function cloneStats(stats) {
  return { ...stats };
}
function updateStats(stats, success, durationMs) {
  stats.attempted++;
  if (success) {
    stats.successful++;
  } else {
    stats.failed++;
  }
  const totalAttempts = stats.successful + stats.failed;
  stats.avgInjectionTimeMs = (stats.avgInjectionTimeMs * (totalAttempts - 1) + durationMs) / totalAttempts;
  stats.lastInjectionAt = (/* @__PURE__ */ new Date()).toISOString();
}
function resetStats(stats) {
  stats.attempted = 0;
  stats.successful = 0;
  stats.failed = 0;
  stats.avgInjectionTimeMs = 0;
  stats.lastInjectionAt = null;
}

// src/recording/injection/strategies/init-script-injection.ts
var InitScriptInjectionStrategy = class {
  name = "init-script";
  initialized = false;
  options = null;
  stats = createInitialStats();
  firstInjectionFired = false;
  /**
   * Initialize the strategy on a browser context.
   *
   * Registers the recording init script using `context.addInitScript()`.
   * The script will run on every new document created in this context.
   *
   * @param context - Browser context to initialize on
   * @param options - Configuration options
   */
  async initialize(context, options) {
    if (this.initialized) {
      options.logger.debug(
        scopedLog(LogContext.INJECTION, "init-script strategy already initialized, skipping")
      );
      return;
    }
    this.options = options;
    const { bindingName, logger: logger2, diagnosticsEnabled } = options;
    const initScript = generateRecordingInitScript(bindingName);
    if (diagnosticsEnabled) {
      logger2.debug(scopedLog(LogContext.INJECTION, "init-script strategy: registering init script"), {
        bindingName,
        scriptLength: initScript.length
      });
    }
    await context.addInitScript(initScript);
    context.on("page", (page) => {
      this.handlePageCreated(page).catch((err) => {
        logger2.warn(scopedLog(LogContext.INJECTION, "error handling page creation"), {
          error: err instanceof Error ? err.message : String(err)
        });
      });
    });
    this.initialized = true;
    logger2.info(scopedLog(LogContext.INJECTION, "init-script strategy initialized"), {
      bindingName
    });
  }
  /**
   * Handle page creation for tracking first injection.
   */
  async handlePageCreated(page) {
    if (!this.options) return;
    const { logger: logger2, diagnosticsEnabled, onFirstInjection } = this.options;
    await page.waitForLoadState("domcontentloaded").catch(() => {
    });
    const startTime = Date.now();
    try {
      const verification = await verifyScriptInjection(page);
      const durationMs = Date.now() - startTime;
      const success = verification.loaded && verification.ready;
      updateStats(this.stats, success, durationMs);
      if (diagnosticsEnabled) {
        logger2.debug(scopedLog(LogContext.INJECTION, "init-script injection result"), {
          url: page.url().slice(0, 80),
          success,
          verification: {
            loaded: verification.loaded,
            ready: verification.ready,
            inMainContext: verification.inMainContext,
            handlersCount: verification.handlersCount
          },
          durationMs
        });
      }
      if (success && !this.firstInjectionFired && onFirstInjection) {
        this.firstInjectionFired = true;
        setImmediate(() => {
          try {
            onFirstInjection();
          } catch (err) {
            logger2.error(scopedLog(LogContext.INJECTION, "first injection callback error"), {
              error: err instanceof Error ? err.message : String(err)
            });
          }
        });
      }
    } catch (error) {
      const durationMs = Date.now() - startTime;
      updateStats(this.stats, false, durationMs);
      logger2.warn(scopedLog(LogContext.INJECTION, "init-script verification failed"), {
        url: page.url().slice(0, 80),
        error: error instanceof Error ? error.message : String(error),
        durationMs
      });
    }
  }
  /**
   * Inject script into a page.
   *
   * For init-script strategy, this is largely a no-op since the script is
   * automatically injected at context level. This method exists for interface
   * compliance and may be used to verify injection after navigation.
   *
   * @param page - The page to inject into
   * @param _script - The script (unused - already registered at context level)
   * @returns Injection result
   */
  async injectScript(page, _script) {
    if (!this.initialized || !this.options) {
      return {
        success: false,
        strategy: this.name,
        error: "Strategy not initialized",
        timestamp: (/* @__PURE__ */ new Date()).toISOString()
      };
    }
    const startTime = Date.now();
    try {
      const verification = await verifyScriptInjection(page);
      const durationMs = Date.now() - startTime;
      const success = verification.loaded && verification.ready;
      updateStats(this.stats, success, durationMs);
      return {
        success,
        strategy: this.name,
        timestamp: (/* @__PURE__ */ new Date()).toISOString(),
        error: success ? void 0 : `Script not ready: loaded=${verification.loaded}, ready=${verification.ready}`,
        metadata: {
          verification: {
            loaded: verification.loaded,
            ready: verification.ready,
            inMainContext: verification.inMainContext,
            handlersCount: verification.handlersCount
          },
          durationMs
        }
      };
    } catch (error) {
      const durationMs = Date.now() - startTime;
      updateStats(this.stats, false, durationMs);
      return {
        success: false,
        strategy: this.name,
        error: error instanceof Error ? error.message : String(error),
        timestamp: (/* @__PURE__ */ new Date()).toISOString(),
        metadata: { durationMs }
      };
    }
  }
  /**
   * Verify that injection was successful on a page.
   *
   * @param page - The page to verify
   * @returns True if script is loaded, ready, and in MAIN context
   */
  async verify(page) {
    try {
      const verification = await verifyScriptInjection(page);
      return verification.loaded && verification.ready && verification.inMainContext;
    } catch {
      return false;
    }
  }
  /**
   * Get current statistics for this strategy.
   */
  getStats() {
    return cloneStats(this.stats);
  }
  /**
   * Reset statistics to initial values.
   */
  resetStats() {
    resetStats(this.stats);
  }
  /**
   * Clean up resources.
   *
   * For init-script strategy, there's nothing to clean up since
   * `addInitScript()` cannot be removed. The script remains registered
   * for the lifetime of the context.
   */
  cleanup() {
    this.options = null;
    this.initialized = false;
    this.firstInjectionFired = false;
    return Promise.resolve();
  }
  /**
   * Check if this strategy supports a given provider.
   *
   * Init-script strategy works with all Playwright providers including
   * rebrowser-playwright. It's the RECOMMENDED strategy for rebrowser.
   *
   * @param _providerName - Name of the provider
   * @returns Always true - works with all providers
   */
  supportsProvider(_providerName) {
    return true;
  }
};

// src/recording/injection/strategies/cdp-injection.ts
var CDPInjectionStrategy = class {
  name = "cdp-injection";
  initialized = false;
  options = null;
  stats = createInitialStats();
  firstInjectionFired = false;
  sessions = [];
  initScript = "";
  /**
   * Initialize the strategy on a browser context.
   *
   * For CDP strategy, we set up listeners for new pages and prepare
   * the injection script. Actual injection happens per-page.
   *
   * @param context - Browser context to initialize on
   * @param options - Configuration options
   */
  async initialize(context, options) {
    if (this.initialized) {
      options.logger.debug(
        scopedLog(LogContext.INJECTION, "cdp-injection strategy already initialized, skipping")
      );
      return;
    }
    this.options = options;
    const { bindingName, logger: logger2, diagnosticsEnabled } = options;
    this.initScript = generateRecordingInitScript(bindingName);
    if (diagnosticsEnabled) {
      logger2.debug(scopedLog(LogContext.INJECTION, "cdp-injection strategy: preparing"), {
        bindingName,
        scriptLength: this.initScript.length
      });
    }
    for (const page of context.pages()) {
      await this.setupPageInjection(page).catch((err) => {
        logger2.warn(scopedLog(LogContext.INJECTION, "failed to setup CDP injection for existing page"), {
          url: page.url().slice(0, 80),
          error: err instanceof Error ? err.message : String(err)
        });
      });
    }
    context.on("page", (page) => {
      this.setupPageInjection(page).catch((err) => {
        logger2.warn(scopedLog(LogContext.INJECTION, "failed to setup CDP injection for new page"), {
          url: page.url().slice(0, 80),
          error: err instanceof Error ? err.message : String(err)
        });
      });
    });
    this.initialized = true;
    logger2.info(scopedLog(LogContext.INJECTION, "cdp-injection strategy initialized"), {
      bindingName
    });
  }
  /**
   * Set up CDP injection for a specific page.
   */
  async setupPageInjection(page) {
    if (!this.options) return;
    const { logger: logger2, diagnosticsEnabled, onFirstInjection } = this.options;
    const startTime = Date.now();
    try {
      const session = await page.context().newCDPSession(page);
      const response = await session.send("Page.addScriptToEvaluateOnNewDocument", {
        source: this.initScript,
        worldName: void 0,
        // Use main world (default)
        runImmediately: true
        // Run immediately, not just on navigation
      });
      this.sessions.push({
        page,
        session,
        scriptIdentifier: response.identifier
      });
      if (diagnosticsEnabled) {
        logger2.debug(scopedLog(LogContext.INJECTION, "cdp-injection: script registered"), {
          url: page.url().slice(0, 80),
          identifier: response.identifier
        });
      }
      const currentUrl = page.url();
      if (currentUrl && currentUrl !== "about:blank") {
        await this.injectIntoExistingPage(page, session);
      }
      page.on("domcontentloaded", async () => {
        await this.handlePageLoad(page, startTime, onFirstInjection);
      });
    } catch (error) {
      const durationMs = Date.now() - startTime;
      updateStats(this.stats, false, durationMs);
      logger2.error(scopedLog(LogContext.INJECTION, "cdp-injection setup failed"), {
        url: page.url().slice(0, 80),
        error: error instanceof Error ? error.message : String(error)
      });
      throw error;
    }
  }
  /**
   * Inject into a page that already has content loaded.
   */
  async injectIntoExistingPage(page, session) {
    if (!this.options) return;
    const { logger: logger2, diagnosticsEnabled } = this.options;
    try {
      await session.send("Runtime.evaluate", {
        expression: this.initScript,
        awaitPromise: false,
        returnByValue: false
      });
      if (diagnosticsEnabled) {
        logger2.debug(scopedLog(LogContext.INJECTION, "cdp-injection: injected into existing page"), {
          url: page.url().slice(0, 80)
        });
      }
    } catch (error) {
      logger2.warn(scopedLog(LogContext.INJECTION, "cdp-injection: failed to inject into existing page"), {
        url: page.url().slice(0, 80),
        error: error instanceof Error ? error.message : String(error)
      });
    }
  }
  /**
   * Handle page load event to track injection success.
   */
  async handlePageLoad(page, startTime, onFirstInjection) {
    if (!this.options) return;
    const { logger: logger2, diagnosticsEnabled } = this.options;
    try {
      const verification = await verifyScriptInjection(page);
      const durationMs = Date.now() - startTime;
      const success = verification.loaded && verification.ready;
      updateStats(this.stats, success, durationMs);
      if (diagnosticsEnabled) {
        logger2.debug(scopedLog(LogContext.INJECTION, "cdp-injection result"), {
          url: page.url().slice(0, 80),
          success,
          verification: {
            loaded: verification.loaded,
            ready: verification.ready,
            inMainContext: verification.inMainContext,
            handlersCount: verification.handlersCount
          },
          durationMs
        });
      }
      if (success && !this.firstInjectionFired && onFirstInjection) {
        this.firstInjectionFired = true;
        setImmediate(() => {
          try {
            onFirstInjection();
          } catch (err) {
            logger2.error(scopedLog(LogContext.INJECTION, "first injection callback error"), {
              error: err instanceof Error ? err.message : String(err)
            });
          }
        });
      }
    } catch (error) {
      logger2.warn(scopedLog(LogContext.INJECTION, "cdp-injection verification failed"), {
        url: page.url().slice(0, 80),
        error: error instanceof Error ? error.message : String(error)
      });
    }
  }
  /**
   * Inject script into a page.
   *
   * For CDP strategy, this executes the script directly in the page
   * using Runtime.evaluate.
   *
   * @param page - The page to inject into
   * @param script - The script to inject (defaults to recorded init script)
   * @returns Injection result
   */
  async injectScript(page, script) {
    if (!this.initialized || !this.options) {
      return {
        success: false,
        strategy: this.name,
        error: "Strategy not initialized",
        timestamp: (/* @__PURE__ */ new Date()).toISOString()
      };
    }
    const scriptToInject = script || this.initScript;
    const startTime = Date.now();
    try {
      const session = await page.context().newCDPSession(page);
      try {
        await session.send("Runtime.evaluate", {
          expression: scriptToInject,
          awaitPromise: false,
          returnByValue: false
        });
        const verification = await verifyScriptInjection(page);
        const durationMs = Date.now() - startTime;
        const success = verification.loaded && verification.ready;
        updateStats(this.stats, success, durationMs);
        return {
          success,
          strategy: this.name,
          timestamp: (/* @__PURE__ */ new Date()).toISOString(),
          error: success ? void 0 : `Script not ready: loaded=${verification.loaded}, ready=${verification.ready}`,
          metadata: {
            verification: {
              loaded: verification.loaded,
              ready: verification.ready,
              inMainContext: verification.inMainContext,
              handlersCount: verification.handlersCount
            },
            durationMs
          }
        };
      } finally {
        await session.detach().catch(() => {
        });
      }
    } catch (error) {
      const durationMs = Date.now() - startTime;
      updateStats(this.stats, false, durationMs);
      return {
        success: false,
        strategy: this.name,
        error: error instanceof Error ? error.message : String(error),
        timestamp: (/* @__PURE__ */ new Date()).toISOString(),
        metadata: { durationMs }
      };
    }
  }
  /**
   * Verify that injection was successful on a page.
   *
   * @param page - The page to verify
   * @returns True if script is loaded, ready, and in MAIN context
   */
  async verify(page) {
    try {
      const verification = await verifyScriptInjection(page);
      return verification.loaded && verification.ready && verification.inMainContext;
    } catch {
      return false;
    }
  }
  /**
   * Get current statistics for this strategy.
   */
  getStats() {
    return cloneStats(this.stats);
  }
  /**
   * Reset statistics to initial values.
   */
  resetStats() {
    resetStats(this.stats);
  }
  /**
   * Clean up resources.
   *
   * Detaches all CDP sessions and removes script registrations.
   */
  async cleanup() {
    const { logger: logger2 } = this.options || {};
    for (const tracker of this.sessions) {
      try {
        if (tracker.scriptIdentifier) {
          await tracker.session.send("Page.removeScriptToEvaluateOnNewDocument", {
            identifier: tracker.scriptIdentifier
          }).catch(() => {
          });
        }
        await tracker.session.detach().catch(() => {
        });
      } catch (error) {
        logger2?.warn(scopedLog(LogContext.INJECTION, "error cleaning up CDP session"), {
          error: error instanceof Error ? error.message : String(error)
        });
      }
    }
    this.sessions = [];
    this.options = null;
    this.initialized = false;
    this.firstInjectionFired = false;
    this.initScript = "";
  }
  /**
   * Check if this strategy supports a given provider.
   *
   * CDP strategy only works with Chromium-based browsers.
   *
   * @param providerName - Name of the provider
   * @returns True if provider is Chromium-based
   */
  supportsProvider(providerName) {
    const chromiumProviders = [
      "rebrowser-playwright",
      "playwright",
      "chromium"
    ];
    const lowerName = providerName.toLowerCase();
    if (lowerName.includes("firefox") || lowerName.includes("webkit")) {
      return false;
    }
    return chromiumProviders.some((p) => lowerName.includes(p));
  }
};

// src/recording/io/html-injector.ts
function createInjectionStats() {
  return {
    attempted: 0,
    successful: 0,
    failed: 0,
    skipped: 0,
    total: 0,
    methods: {
      head: 0,
      HEAD: 0,
      doctype: 0,
      prepend: 0
    }
  };
}
function cloneInjectionStats(stats) {
  return {
    ...stats,
    methods: { ...stats.methods }
  };
}
function resetInjectionStats(stats) {
  stats.attempted = 0;
  stats.successful = 0;
  stats.failed = 0;
  stats.skipped = 0;
  stats.total = 0;
  stats.methods.head = 0;
  stats.methods.HEAD = 0;
  stats.methods.doctype = 0;
  stats.methods.prepend = 0;
}
function injectScriptIntoHtml(originalBody, initScript) {
  const scriptTag = `<script>${initScript}</script>`;
  let method;
  let modifiedBody;
  if (originalBody.includes("<head>")) {
    modifiedBody = originalBody.replace("<head>", `<head>${scriptTag}`);
    method = "head";
  } else if (originalBody.includes("<HEAD>")) {
    modifiedBody = originalBody.replace("<HEAD>", `<HEAD>${scriptTag}`);
    method = "HEAD";
  } else if (originalBody.toLowerCase().includes("<!doctype")) {
    modifiedBody = originalBody.replace(/<!doctype[^>]*>/i, (match) => `${match}${scriptTag}`);
    method = "doctype";
  } else {
    modifiedBody = scriptTag + originalBody;
    method = "prepend";
  }
  return {
    modifiedBody,
    method,
    originalLength: originalBody.length,
    modifiedLength: modifiedBody.length
  };
}
async function handleRouteForInjection(route, request, initScript, stats, logger2, diagnosticsEnabled) {
  const url = request.url();
  const resourceType = request.resourceType();
  const preFetchDecision = shouldInjectScript(resourceType, url);
  if (!preFetchDecision.shouldInject) {
    if (diagnosticsEnabled || preFetchDecision.reason === "skip_test_page") {
      logger2.debug(
        scopedLog(LogContext.INJECTION, `decision: ${preFetchDecision.reason}`),
        formatDecisionForLog(preFetchDecision)
      );
    }
    if (preFetchDecision.reason === "skip_test_page") {
      await route.fallback();
    } else {
      stats.skipped++;
      await route.continue();
    }
    return false;
  }
  stats.attempted++;
  stats.total++;
  if (diagnosticsEnabled) {
    logger2.debug(scopedLog(LogContext.INJECTION, "intercepting document request"), {
      url: url.slice(0, 80),
      resourceType,
      attemptNumber: stats.attempted
    });
  }
  try {
    logger2.info(scopedLog(LogContext.INJECTION, "fetching document for injection"), {
      url: url.slice(0, 100),
      resourceType
    });
    const response = await route.fetch({
      maxRedirects: 10,
      // Follow redirects to get final content
      timeout: 3e4
      // 30s timeout to prevent hanging
    });
    const status = response.status();
    const contentType = response.headers()["content-type"] || "";
    logger2.info(scopedLog(LogContext.INJECTION, "document fetched"), {
      url: url.slice(0, 100),
      status,
      contentType: contentType.slice(0, 50)
    });
    const postFetchDecision = shouldInjectScript(resourceType, url, status, contentType);
    if (!postFetchDecision.shouldInject) {
      logger2.debug(
        scopedLog(LogContext.INJECTION, `decision: ${postFetchDecision.reason}`),
        formatDecisionForLog(postFetchDecision)
      );
      stats.skipped++;
      await route.fulfill({ response });
      return false;
    }
    const originalBody = await response.text();
    const injection = injectScriptIntoHtml(originalBody, initScript);
    logger2.info(scopedLog(LogContext.INJECTION, "HTML modified for injection"), {
      url: url.slice(0, 100),
      method: injection.method,
      originalLength: injection.originalLength,
      modifiedLength: injection.modifiedLength,
      scriptInjectedLength: injection.modifiedLength - injection.originalLength
    });
    await route.fulfill({
      response,
      body: injection.modifiedBody
    });
    stats.successful++;
    stats.methods[injection.method]++;
    logger2.info(scopedLog(LogContext.INJECTION, "route.fulfill completed - injection successful"), {
      url: url.slice(0, 100),
      method: injection.method,
      stats: cloneInjectionStats(stats)
    });
    return true;
  } catch (error) {
    stats.failed++;
    logger2.error(scopedLog(LogContext.INJECTION, "injection failed"), {
      url: request.url().slice(0, 80),
      error: error instanceof Error ? error.message : String(error),
      stats: cloneInjectionStats(stats)
    });
    await route.continue();
    return false;
  }
}
async function setupHtmlInjectionRoute(context, options) {
  const { bindingName, logger: logger2, diagnosticsEnabled = false, onFirstInjection } = options;
  const initScript = generateRecordingInitScript(bindingName);
  const stats = createInjectionStats();
  let firstInjectionFired = false;
  await context.route("**/*", async (route) => {
    const request = route.request();
    const success = await handleRouteForInjection(
      route,
      request,
      initScript,
      stats,
      logger2,
      diagnosticsEnabled
    );
    if (success && !firstInjectionFired && onFirstInjection) {
      firstInjectionFired = true;
      setImmediate(() => {
        try {
          onFirstInjection();
        } catch (err) {
          logger2.error(scopedLog(LogContext.INJECTION, "first injection callback error"), {
            error: err instanceof Error ? err.message : String(err)
          });
        }
      });
    }
  });
  logger2.info(scopedLog(LogContext.RECORDING, "HTML injection route set up"));
  return {
    getStats: () => cloneInjectionStats(stats),
    resetStats: () => resetInjectionStats(stats)
  };
}

// src/recording/injection/strategies/route-injection.ts
var RouteInjectionStrategy = class {
  name = "route-injection";
  initialized = false;
  options = null;
  stats = createInitialStats();
  getHtmlStats = null;
  resetHtmlStats = null;
  /**
   * Initialize the strategy on a browser context.
   *
   * Sets up the HTML injection route handler using the existing
   * `setupHtmlInjectionRoute()` function.
   *
   * @param context - Browser context to initialize on
   * @param options - Configuration options
   */
  async initialize(context, options) {
    if (this.initialized) {
      options.logger.debug(
        scopedLog(LogContext.INJECTION, "route-injection strategy already initialized, skipping")
      );
      return;
    }
    this.options = options;
    const { bindingName, logger: logger2, diagnosticsEnabled, onFirstInjection } = options;
    logger2.warn(
      scopedLog(
        LogContext.INJECTION,
        "route-injection strategy is DEPRECATED for rebrowser-playwright. Route interception is broken with anti-detection patches. Use init-script strategy instead."
      )
    );
    if (diagnosticsEnabled) {
      logger2.debug(scopedLog(LogContext.INJECTION, "route-injection strategy: setting up"), {
        bindingName
      });
    }
    const result = await setupHtmlInjectionRoute(context, {
      bindingName,
      logger: logger2,
      diagnosticsEnabled,
      onFirstInjection: () => {
        this.syncStatsFromHtmlInjector();
        if (onFirstInjection) {
          onFirstInjection();
        }
      }
    });
    this.getHtmlStats = result.getStats;
    this.resetHtmlStats = result.resetStats;
    this.initialized = true;
    logger2.info(scopedLog(LogContext.INJECTION, "route-injection strategy initialized"), {
      bindingName
    });
  }
  /**
   * Sync stats from the HTML injector to our stats format.
   */
  syncStatsFromHtmlInjector() {
    if (!this.getHtmlStats) return;
    const htmlStats = this.getHtmlStats();
    this.stats.attempted = htmlStats.attempted;
    this.stats.successful = htmlStats.successful;
    this.stats.failed = htmlStats.failed;
    this.stats.lastInjectionAt = (/* @__PURE__ */ new Date()).toISOString();
    if (htmlStats.successful > 0) {
      this.stats.avgInjectionTimeMs = 50;
    }
  }
  /**
   * Inject script into a page.
   *
   * For route-injection, this is a no-op since injection happens automatically
   * via the route handler when pages navigate. This method verifies the injection.
   *
   * @param page - The page to inject into
   * @param _script - The script (unused - handled by route handler)
   * @returns Injection result
   */
  async injectScript(page, _script) {
    if (!this.initialized || !this.options) {
      return {
        success: false,
        strategy: this.name,
        error: "Strategy not initialized",
        timestamp: (/* @__PURE__ */ new Date()).toISOString()
      };
    }
    const startTime = Date.now();
    try {
      this.syncStatsFromHtmlInjector();
      const verification = await verifyScriptInjection(page);
      const durationMs = Date.now() - startTime;
      const success = verification.loaded && verification.ready;
      return {
        success,
        strategy: this.name,
        timestamp: (/* @__PURE__ */ new Date()).toISOString(),
        error: success ? void 0 : `Script not ready: loaded=${verification.loaded}, ready=${verification.ready}`,
        metadata: {
          verification: {
            loaded: verification.loaded,
            ready: verification.ready,
            inMainContext: verification.inMainContext,
            handlersCount: verification.handlersCount
          },
          htmlInjectorStats: this.getHtmlStats?.() ?? null,
          durationMs
        }
      };
    } catch (error) {
      return {
        success: false,
        strategy: this.name,
        error: error instanceof Error ? error.message : String(error),
        timestamp: (/* @__PURE__ */ new Date()).toISOString(),
        metadata: { durationMs: Date.now() - startTime }
      };
    }
  }
  /**
   * Verify that injection was successful on a page.
   *
   * @param page - The page to verify
   * @returns True if script is loaded, ready, and in MAIN context
   */
  async verify(page) {
    try {
      const verification = await verifyScriptInjection(page);
      return verification.loaded && verification.ready && verification.inMainContext;
    } catch {
      return false;
    }
  }
  /**
   * Get current statistics for this strategy.
   */
  getStats() {
    this.syncStatsFromHtmlInjector();
    return cloneStats(this.stats);
  }
  /**
   * Reset statistics to initial values.
   */
  resetStats() {
    resetStats(this.stats);
    if (this.resetHtmlStats) {
      this.resetHtmlStats();
    }
  }
  /**
   * Clean up resources.
   *
   * Note: Route handlers registered with context.route() cannot be removed
   * in Playwright. The handler will remain active for the context lifetime.
   */
  cleanup() {
    this.options = null;
    this.initialized = false;
    this.getHtmlStats = null;
    this.resetHtmlStats = null;
    return Promise.resolve();
  }
  /**
   * Check if this strategy supports a given provider.
   *
   * Route injection only works reliably with standard Playwright.
   * It is BROKEN with rebrowser-playwright.
   *
   * @param providerName - Name of the provider
   * @returns True only for standard Playwright (not rebrowser)
   */
  supportsProvider(providerName) {
    const lowerName = providerName.toLowerCase();
    if (lowerName.includes("rebrowser")) {
      return false;
    }
    return lowerName.includes("playwright");
  }
};

// src/recording/injection/factory.ts
var INJECTION_STRATEGY_ENV_VAR = "INJECTION_STRATEGY";
var INJECTION_DIAGNOSTICS_ENV_VAR = "INJECTION_DIAGNOSTICS";
function getStrategyFromEnv() {
  const envValue = process.env[INJECTION_STRATEGY_ENV_VAR];
  if (!envValue) return null;
  const normalizedValue = envValue.toLowerCase().trim();
  if (normalizedValue === "auto") return "auto";
  if (normalizedValue === "init-script" || normalizedValue === "initscript") return "init-script";
  if (normalizedValue === "cdp-injection" || normalizedValue === "cdp") return "cdp-injection";
  if (normalizedValue === "route-injection" || normalizedValue === "route") return "route-injection";
  return null;
}
function isDiagnosticsEnabled() {
  const envValue = process.env[INJECTION_DIAGNOSTICS_ENV_VAR];
  return envValue === "true" || envValue === "1";
}
function selectStrategyForProvider(providerName) {
  const lowerName = providerName.toLowerCase();
  if (lowerName.includes("rebrowser")) {
    return "init-script";
  }
  return "init-script";
}
var InjectionStrategyFactory = class {
  logger;
  constructor(logger2) {
    this.logger = logger2 ?? logger;
  }
  /**
   * Create an injection strategy based on options.
   *
   * Selection priority:
   * 1. INJECTION_STRATEGY environment variable
   * 2. Explicit strategyName option
   * 3. Provider-based auto-selection
   *
   * @param options - Factory options
   * @returns Created strategy instance
   */
  create(options = {}) {
    const envStrategy = getStrategyFromEnv();
    if (envStrategy && envStrategy !== "auto") {
      this.logger.info(scopedLog(LogContext.INJECTION, "using strategy from environment variable"), {
        strategy: envStrategy,
        envVar: INJECTION_STRATEGY_ENV_VAR
      });
      return this.createByName(envStrategy);
    }
    const requestedStrategy = options.strategyName;
    if (requestedStrategy && requestedStrategy !== "auto") {
      this.logger.info(scopedLog(LogContext.INJECTION, "using explicitly requested strategy"), {
        strategy: requestedStrategy
      });
      return this.createByName(requestedStrategy);
    }
    const providerName = options.providerName ?? "rebrowser-playwright";
    const selectedStrategy = selectStrategyForProvider(providerName);
    this.logger.info(scopedLog(LogContext.INJECTION, "auto-selected strategy for provider"), {
      strategy: selectedStrategy,
      provider: providerName
    });
    return this.createByName(selectedStrategy);
  }
  /**
   * Create a strategy by name.
   *
   * @param name - Strategy name
   * @returns Strategy instance
   * @throws Error if strategy name is unknown
   */
  createByName(name) {
    switch (name) {
      case "init-script":
        return new InitScriptInjectionStrategy();
      case "cdp-injection":
        return new CDPInjectionStrategy();
      case "route-injection":
        return new RouteInjectionStrategy();
      default: {
        const unknownName = String(name);
        throw new Error(`Unknown injection strategy: ${unknownName}`);
      }
    }
  }
  /**
   * Get all available strategy names.
   */
  getAvailableStrategies() {
    return ["init-script", "cdp-injection", "route-injection"];
  }
  /**
   * Check if a strategy supports a given provider.
   *
   * @param strategyName - Strategy to check
   * @param providerName - Provider to check against
   * @returns True if strategy supports the provider
   */
  strategySupportsProvider(strategyName, providerName) {
    const strategy = this.createByName(strategyName);
    return strategy.supportsProvider(providerName);
  }
};
function createInjectionStrategy(options = {}) {
  const factory = new InjectionStrategyFactory(options.logger);
  return factory.create(options);
}

// src/recording/io/context-initializer.ts
var RecordingContextInitializer = class {
  initialized = false;
  eventHandler = null;
  bindingName;
  logger;
  diagnosticsEnabled;
  runSanityCheck;
  onSanityCheckComplete;
  requestedStrategy;
  sanityCheckRun = false;
  context = null;
  // Composed modules
  eventRouteManager = null;
  // Injection strategy (DI system)
  injectionStrategy = null;
  constructor(options = {}) {
    this.bindingName = options.bindingName ?? DEFAULT_RECORDING_BINDING_NAME;
    this.logger = options.logger ?? logger;
    this.diagnosticsEnabled = options.diagnosticsEnabled ?? isDiagnosticsEnabled();
    this.runSanityCheck = options.runSanityCheck ?? false;
    this.onSanityCheckComplete = options.onSanityCheckComplete;
    this.requestedStrategy = options.injectionStrategy ?? "auto";
  }
  /**
   * Get current injection statistics.
   * Returns a copy to prevent external mutation.
   */
  getInjectionStats() {
    if (this.injectionStrategy) {
      return this.injectionStrategy.getStats();
    }
    return createInitialStats();
  }
  /**
   * Get the name of the injection strategy being used.
   */
  getInjectionStrategyName() {
    return this.injectionStrategy?.name ?? null;
  }
  /**
   * Get current route handler statistics.
   * Returns a copy to prevent external mutation.
   */
  getRouteHandlerStats() {
    if (this.eventRouteManager) {
      return this.eventRouteManager.getStats();
    }
    return createRouteHandlerStats();
  }
  /**
   * Reset all stats (injection and route handler) to initial values.
   * Useful for clearing state between test runs to prevent cumulative stats.
   *
   * This ensures consistent test results by starting from a clean slate.
   */
  resetStats() {
    if (this.injectionStrategy) {
      this.injectionStrategy.resetStats();
    }
    if (this.eventRouteManager) {
      this.eventRouteManager.resetStats();
    }
    this.logger.debug(scopedLog(LogContext.RECORDING, "stats reset to initial values"));
  }
  /**
   * Check if an event handler is currently set.
   * Useful for diagnostics to verify the event pipeline is connected.
   */
  hasEventHandler() {
    return this.eventHandler !== null;
  }
  /**
   * Setup page-level route for event interception.
   *
   * Delegates to the event route manager.
   *
   * @param page - The page to setup event interception on
   * @param options - Options for route setup
   */
  async setupPageEventRoute(page, options = {}) {
    if (!this.eventRouteManager) {
      throw new Error("Context not initialized. Call initialize() first.");
    }
    await this.eventRouteManager.setupPageEventRoute(page, options);
  }
  /**
   * Initialize recording capability on a browser context.
   *
   * This sets up:
   * 1. Injection strategy (injects recording script into pages)
   * 2. The event route manager (handles events from pages)
   *
   * Safe to call multiple times (idempotent).
   *
   * @param context - The browser context to initialize
   */
  async initialize(context) {
    if (this.initialized) {
      this.logger.debug(scopedLog(LogContext.RECORDING, "context already initialized, skipping"));
      return;
    }
    const envStrategy = getStrategyFromEnv();
    const strategyToUse = envStrategy ?? this.requestedStrategy;
    this.logger.debug(scopedLog(LogContext.RECORDING, "initializing recording context"), {
      bindingName: this.bindingName,
      runSanityCheck: this.runSanityCheck,
      requestedStrategy: this.requestedStrategy,
      strategyToUse
    });
    this.context = context;
    this.eventRouteManager = createEventRouteManager({
      logger: this.logger,
      getEventHandler: () => this.eventHandler
    });
    this.injectionStrategy = createInjectionStrategy({
      strategyName: strategyToUse,
      providerName: playwrightProvider.name,
      logger: this.logger
    });
    this.logger.info(scopedLog(LogContext.INJECTION, "using injection strategy"), {
      strategy: this.injectionStrategy.name,
      provider: playwrightProvider.name
    });
    await this.injectionStrategy.initialize(context, {
      bindingName: this.bindingName,
      logger: this.logger,
      diagnosticsEnabled: this.diagnosticsEnabled,
      onFirstInjection: () => {
        if (this.runSanityCheck && !this.sanityCheckRun) {
          this.triggerSanityCheck().catch((err) => {
            this.logger.error(scopedLog(LogContext.RECORDING, "sanity check failed"), {
              error: err instanceof Error ? err.message : String(err)
            });
          });
        }
      }
    });
    const existingPages = context.pages();
    for (const page of existingPages) {
      try {
        await this.eventRouteManager.setupPageEventRoute(page);
      } catch (err) {
        this.logger.warn(scopedLog(LogContext.RECORDING, "failed to setup event route for existing page"), {
          url: page.url()?.slice(0, 50),
          error: err instanceof Error ? err.message : String(err)
        });
      }
    }
    context.on("page", async (page) => {
      this.logger.debug(scopedLog(LogContext.RECORDING, "new page created, setting up event route"), {
        url: page.url()?.slice(0, 50)
      });
      const eventRouteManager = this.eventRouteManager;
      if (!eventRouteManager) {
        this.logger.warn(scopedLog(LogContext.RECORDING, "event route manager unavailable for new page"), {
          url: page.url()?.slice(0, 50)
        });
        return;
      }
      try {
        await eventRouteManager.setupPageEventRoute(page);
      } catch (err) {
        this.logger.warn(scopedLog(LogContext.RECORDING, "failed to setup event route for new page"), {
          url: page.url()?.slice(0, 50),
          error: err instanceof Error ? err.message : String(err)
        });
      }
    });
    this.initialized = true;
    this.logger.info(scopedLog(LogContext.RECORDING, "recording context initialized"), {
      bindingName: this.bindingName,
      injectionStrategy: this.injectionStrategy?.name ?? "unknown"
    });
  }
  /**
   * Set the handler for recording events.
   *
   * Called by RecordingPipelineManager when recording starts.
   * Only one handler can be active at a time.
   *
   * @param handler - Function to receive recording events
   */
  setEventHandler(handler) {
    this.eventHandler = handler;
    this.logger.debug(scopedLog(LogContext.RECORDING, "event handler set"));
  }
  /**
   * Clear the event handler.
   *
   * Called when recording stops. Events will be silently dropped
   * until a new handler is set.
   */
  clearEventHandler() {
    this.eventHandler = null;
    this.logger.debug(scopedLog(LogContext.RECORDING, "event handler cleared"));
  }
  /**
   * Check if this context has been initialized.
   */
  isInitialized() {
    return this.initialized;
  }
  /**
   * Get the binding name used for this initializer.
   */
  getBindingName() {
    return this.bindingName;
  }
  // ===========================================================================
  // Sanity Check Logic
  // ===========================================================================
  /**
   * Run a sanity check on a page to verify recording is working.
   *
   * This can be called manually or is run automatically on first page load
   * if runSanityCheck option is enabled.
   *
   * The sanity check verifies:
   * 1. Script was injected (loaded marker set)
   * 2. Script initialized successfully (ready marker set)
   * 3. Script is running in MAIN context (required for History API)
   * 4. Handlers are registered (events will be captured)
   *
   * @param page - The page to check
   * @returns Sanity check result
   */
  async runSanityCheckOnPage(page) {
    const startTime = Date.now();
    const url = page.url();
    const issues = [];
    this.logger.debug(scopedLog(LogContext.RECORDING, "running sanity check"), { url });
    try {
      const verification = await waitForScriptReady(page, 5e3);
      const result = {
        ready: false,
        timestamp: (/* @__PURE__ */ new Date()).toISOString(),
        url,
        durationMs: Date.now() - startTime,
        scriptVerification: {
          loaded: verification.loaded,
          ready: verification.ready,
          inMainContext: verification.inMainContext,
          handlersCount: verification.handlersCount,
          version: verification.version,
          error: verification.error
        },
        issues,
        provider: playwrightProvider.name,
        injectionStrategy: this.injectionStrategy?.name
      };
      if (!verification.loaded) {
        issues.push(
          `Script not loaded. This likely means HTML injection failed. Check route interception and that the page was navigated via HTTP(S). Error: ${verification.error || "unknown"}`
        );
      } else if (verification.initError) {
        issues.push(
          `Script crashed during initialization: ${verification.initError}. Check browser console for details.`
        );
      } else if (!verification.ready) {
        issues.push(
          `Script loaded but not ready. Handlers registered: ${verification.handlersCount}. Script may have partially initialized before crashing.`
        );
      } else if (!verification.inMainContext) {
        issues.push(
          `Script running in ISOLATED context instead of MAIN. This means History API navigation events will NOT be captured. The script should be injected via HTML, not page.evaluate().`
        );
      } else if (verification.handlersCount < 7) {
        issues.push(
          `Low handler count (${verification.handlersCount}). Expected 7+ handlers. Some event types may not be captured.`
        );
      }
      result.ready = verification.loaded && verification.ready && verification.inMainContext && !verification.initError;
      if (result.ready) {
        this.logger.info(scopedLog(LogContext.RECORDING, "sanity check PASSED"), {
          url: url.slice(0, 80),
          durationMs: result.durationMs,
          handlersCount: verification.handlersCount,
          version: verification.version
        });
      } else {
        this.logger.warn(scopedLog(LogContext.RECORDING, "sanity check FAILED"), {
          url: url.slice(0, 80),
          durationMs: result.durationMs,
          issues,
          verification: result.scriptVerification
        });
      }
      return result;
    } catch (error) {
      const result = {
        ready: false,
        timestamp: (/* @__PURE__ */ new Date()).toISOString(),
        url,
        durationMs: Date.now() - startTime,
        issues: [`Sanity check error: ${error instanceof Error ? error.message : String(error)}`],
        provider: playwrightProvider.name,
        injectionStrategy: this.injectionStrategy?.name
      };
      this.logger.error(scopedLog(LogContext.RECORDING, "sanity check ERROR"), {
        url: url.slice(0, 80),
        error: error instanceof Error ? error.message : String(error),
        injectionStrategy: this.injectionStrategy?.name
      });
      return result;
    }
  }
  /**
   * Internal method to trigger sanity check after first injection.
   * Called from the HTML injector when first successful injection occurs.
   */
  async triggerSanityCheck() {
    if (!this.runSanityCheck || this.sanityCheckRun || !this.context) {
      return;
    }
    this.sanityCheckRun = true;
    const pages = this.context.pages();
    if (pages.length === 0) {
      this.logger.debug(scopedLog(LogContext.RECORDING, "no pages for sanity check"));
      return;
    }
    const page = pages[0];
    if (!page) {
      this.logger.debug(scopedLog(LogContext.RECORDING, "no page available for sanity check"));
      return;
    }
    await new Promise((resolve) => setTimeout(resolve, 500));
    const result = await this.runSanityCheckOnPage(page);
    if (this.onSanityCheckComplete) {
      try {
        this.onSanityCheckComplete(result);
      } catch (error) {
        this.logger.error(scopedLog(LogContext.RECORDING, "sanity check callback error"), {
          error: error instanceof Error ? error.message : String(error)
        });
      }
    }
  }
};
function createRecordingContextInitializer(options = {}) {
  return new RecordingContextInitializer(options);
}

// src/recording/io/buffer.ts
var entryBuffers = /* @__PURE__ */ new Map();
var evictionCounts = /* @__PURE__ */ new Map();
var seenEntryIds = /* @__PURE__ */ new Map();
var lastSequenceNums = /* @__PURE__ */ new Map();
function removeRecordingBuffer(sessionId) {
  entryBuffers.delete(sessionId);
  evictionCounts.delete(sessionId);
  seenEntryIds.delete(sessionId);
  lastSequenceNums.delete(sessionId);
}
registerSessionCleanup("recording-buffer", (sessionId) => {
  removeRecordingBuffer(sessionId);
});

// src/audio-bisect.ts
var WAV = "/home/matthalloran8/Vrooli/scenarios/audio-tools/bas/fixtures/dictation-reference.wav";
var ARGS = [
  "--use-fake-device-for-media-stream",
  "--use-fake-ui-for-media-stream",
  `--use-file-for-fake-audio-capture=${WAV}`,
  "--headless=new",
  "--no-sandbox",
  "--disable-dev-shm-usage",
  "--ozone-platform=headless"
];
var PROBE = async () => {
  const r = {};
  const ac = new AudioContext();
  try {
    await ac.resume();
  } catch (e) {
    r.rErr = String(e);
  }
  r.state = ac.state;
  const t0 = ac.currentTime;
  const sp = ac.createScriptProcessor(4096, 1, 1);
  r.cb = 0;
  r.micMax = 0;
  sp.onaudioprocess = (e) => {
    r.cb++;
    const d = e.inputBuffer.getChannelData(0);
    for (let i = 0; i < d.length; i++) {
      const a = Math.abs(d[i]);
      if (a > r.micMax) r.micMax = a;
    }
  };
  try {
    const s = await navigator.mediaDevices.getUserMedia({ audio: true });
    ac.createMediaStreamSource(s).connect(sp);
  } catch (e) {
    r.gum = String(e).slice(0, 50);
  }
  sp.connect(ac.destination);
  await new Promise((x) => setTimeout(x, 2500));
  r.clock = +(ac.currentTime - t0).toFixed(2);
  return r;
};
async function run(name, apply, opts = {}) {
  let out;
  const b = await import_rebrowser_playwright2.chromium.launch({ headless: false, args: ARGS });
  try {
    const ctx = await b.newContext({ viewport: { width: 1280, height: 720 }, ...opts });
    await apply(ctx);
    const p = await ctx.newPage();
    await p.goto("http://localhost:20004/", { waitUntil: "load", timeout: 2e4 });
    out = await p.evaluate(PROBE);
  } catch (e) {
    out = { fatal: String(e).slice(0, 160) };
  }
  await b.close();
  const broken = !out.cb;
  console.log(`${broken ? "BROKEN " : "ok     "} ${name.padEnd(34)} ${JSON.stringify(out)}`);
}
(async () => {
  await run("A control", async () => {
  });
  await run("B service-worker controller", async (ctx) => {
    const swc = new ServiceWorkerController("bisect", { mode: "allow" });
    await swc.setupBlockingForContext(ctx);
  });
  await run("C recording initializer", async (ctx) => {
    const init = createRecordingContextInitializer({ logger });
    await init.initialize(ctx);
  });
  await run("D second context in browser", async () => {
  });
  process.exit(0);
})();
