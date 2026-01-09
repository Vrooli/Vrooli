// @ts-check

/**
 * Event feed management – logging, transcript tracking, and rendering helpers.
 */

import {
  elements,
  state,
  AGGREGATED_EVENT_TYPES,
  SUPPRESSED_EVENT_LABELS,
} from "./state.js";
import { formatEventPayload } from "./utils.js";

/** @typedef {import("./types.d.ts").TerminalTab} TerminalTab */
/** @typedef {import("./types.d.ts").TranscriptEntry} TranscriptEntry */

/** @type {((reason: string) => void) | null} */
let activeTabMutationHandler = null;

const MAX_TRANSCRIPT_BYTES = 1024 * 1024; // cap transcript cache to roughly 1 MiB per tab

/**
 * @param {TranscriptEntry | null | undefined} entry
 * @returns {number}
 */
function estimateTranscriptEntryBytes(entry) {
  if (!entry || typeof entry !== "object") {
    return 0;
  }
  let size = 0;
  if (typeof entry.data === "string") {
    size += entry.data.length;
  }
  if (typeof entry.message === "string") {
    size += entry.message.length;
  }
  if (typeof entry.direction === "string") {
    size += entry.direction.length;
  }
  return size;
}

/**
 * @param {TerminalTab | null | undefined} tab
 * @returns {void}
 */
function trimTranscript(tab) {
  if (!tab || !Array.isArray(tab.transcript)) {
    return;
  }
  if (!Number.isFinite(tab.transcriptByteSize)) {
    tab.transcriptByteSize = tab.transcript.reduce(
      (total, entry) => total + estimateTranscriptEntryBytes(entry),
      0,
    );
  }
  while (
    (tab.transcriptByteSize > MAX_TRANSCRIPT_BYTES ||
      tab.transcript.length > 5000) &&
    tab.transcript.length > 0
  ) {
    const removed = tab.transcript.shift();
    tab.transcriptByteSize -= estimateTranscriptEntryBytes(removed);
    if (tab.transcriptByteSize < 0) {
      tab.transcriptByteSize = 0;
    }
  }
}

/**
 * @param {{ onActiveTabMutation?: (reason: string) => void }} [options]
 */
export function configureEventFeed({ onActiveTabMutation } = {}) {
  activeTabMutationHandler =
    typeof onActiveTabMutation === "function" ? onActiveTabMutation : null;
}

/**
 * @param {TerminalTab | null | undefined} tab
 * @param {string} type
 * @param {unknown} payload
 * @returns {void}
 */
export function appendEvent(tab, type, payload) {
  if (!tab) return;
  const normalized =
    payload instanceof Error
      ? { message: payload.message, stack: payload.stack }
      : payload;
  if (AGGREGATED_EVENT_TYPES.has(type)) {
    recordSuppressedEvent(tab, type);
    return;
  }
  tab.events.push({ type, payload: normalized, timestamp: Date.now() });
  if (tab.events.length > 500) {
    tab.events.splice(0, tab.events.length - 500);
  }
  if (!state.drawer.open && tab.id === state.activeTabId) {
    const nextCount = (state.drawer.unreadCount || 0) + 1;
    state.drawer.unreadCount = nextCount > 999 ? 999 : nextCount;
  }
  if (tab.id === state.activeTabId && activeTabMutationHandler) {
    activeTabMutationHandler("event-appended");
  }
}

/**
 * @param {TerminalTab | null | undefined} tab
 * @param {string} type
 * @returns {void}
 */
export function recordSuppressedEvent(tab, type) {
  if (!tab) return;
  const current = (tab.suppressed[type] || 0) + 1;
  tab.suppressed[type] = current;
  if (tab.id === state.activeTabId) {
    renderEventMeta(tab);
    if (activeTabMutationHandler) {
      activeTabMutationHandler("suppressed-event");
    }
  }
}

/**
 * @param {TerminalTab | null | undefined} tab
 * @param {TranscriptEntry} entry
 * @returns {void}
 */
export function recordTranscript(tab, entry) {
  if (!tab) return;
  const size = estimateTranscriptEntryBytes(entry);
  if (!Number.isFinite(tab.transcriptByteSize)) {
    tab.transcriptByteSize = 0;
  }
  tab.transcript.push(entry);
  tab.transcriptByteSize += size;
  trimTranscript(tab);
  if (tab.id === state.activeTabId && activeTabMutationHandler) {
    activeTabMutationHandler("transcript-recorded");
  }
}

/**
 * @param {TerminalTab | null | undefined} tab
 * @param {string} message
 * @returns {void}
 */
export function showError(tab, message) {
  if (!tab) return;
  tab.errorMessage = message || "";
  if (tab.id === state.activeTabId && activeTabMutationHandler) {
    activeTabMutationHandler("error-updated");
  }
}

/**
 * @param {TerminalTab | null | undefined} tab
 * @returns {void}
 */
export function renderEvents(tab) {
  const eventFeed = elements.eventFeed;
  if (!eventFeed) return;
  eventFeed.innerHTML = "";
  const events = tab && Array.isArray(tab.events) ? tab.events : [];
  const totalEvents = events.length;
  const eventFeedCount = elements.eventFeedCount;
  if (eventFeedCount) {
    const countLabel =
      totalEvents === 0
        ? "No events recorded"
        : totalEvents === 1
          ? "1 event recorded (latest 50 shown)"
          : `${totalEvents} events recorded (latest 50 shown)`;
    eventFeedCount.textContent = String(totalEvents);
    eventFeedCount.setAttribute("aria-label", countLabel);
    eventFeedCount.title = countLabel;
  }
  if (!tab) return;
  const recent = events.slice(-50).reverse();
  recent.forEach((event) => {
    const li = document.createElement("li");
    const time = document.createElement("time");
    time.textContent = new Date(event.timestamp).toLocaleTimeString();
    const label = document.createElement("div");
    label.textContent = event.type;
    label.style.fontWeight = "600";
    li.appendChild(time);
    li.appendChild(label);
    if (event.payload !== undefined) {
      const detail = document.createElement("pre");
      detail.className = "event-detail";
      detail.textContent = formatEventPayload(event.payload);
      li.appendChild(detail);
    }
    eventFeed.appendChild(li);
  });
}

/**
 * @param {TerminalTab | null | undefined} tab
 * @returns {void}
 */
export function renderEventMeta(tab) {
  const eventMeta = elements.eventMeta;
  if (!eventMeta) return;
  if (!tab) {
    eventMeta.textContent = "";
    eventMeta.classList.add("hidden");
    return;
  }
  const suppressedLabels = /** @type {Record<string, string>} */ (
    SUPPRESSED_EVENT_LABELS
  );
  const summaries = Object.entries(tab.suppressed)
    .filter(([, count]) => count > 0)
    .map(
      ([type, count]) => `${suppressedLabels[type] || type}: ${count}`,
    );

  if (summaries.length === 0) {
    eventMeta.textContent = "";
    eventMeta.classList.add("hidden");
  } else {
    eventMeta.textContent = summaries.join(" • ");
    eventMeta.classList.remove("hidden");
  }
}

/**
 * @param {TerminalTab | null | undefined} tab
 * @returns {void}
 */
export function renderError(tab) {
  const errorBanner = elements.errorBanner;
  if (!errorBanner) return;
  const message = tab ? tab.errorMessage : "";
  if (message) {
    errorBanner.textContent = message;
    errorBanner.classList.remove("hidden");
  } else {
    errorBanner.textContent = "";
    errorBanner.classList.add("hidden");
  }
}
