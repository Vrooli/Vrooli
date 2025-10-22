import { elements, state } from "../state.js";
import { proxyToApi, parseApiError } from "../utils.js";
import { showToast } from "../notifications.js";
import {
  IDLE_TIMEOUT_DEFAULT_MINUTES,
  IDLE_TIMEOUT_MAX_MINUTES,
} from "./constants.js";

export function setWorkspaceLoading(value) {
  if (!state.workspace) {
    state.workspace = {
      loading: Boolean(value),
      idleTimeoutSeconds: 0,
      updatingIdleTimeout: false,
      idleControlsInitialized: false,
    };
    return;
  }
  state.workspace.loading = Boolean(value);
  if (typeof state.workspace.idleTimeoutSeconds !== "number") {
    state.workspace.idleTimeoutSeconds = 0;
  }
  if (typeof state.workspace.updatingIdleTimeout !== "boolean") {
    state.workspace.updatingIdleTimeout = false;
  }
  if (typeof state.workspace.idleControlsInitialized !== "boolean") {
    state.workspace.idleControlsInitialized = false;
  }
}

export function isWorkspaceLoading() {
  return Boolean(state.workspace && state.workspace.loading);
}

export function applyIdleTimeoutFromServer(seconds) {
  const normalized =
    Number.isFinite(seconds) && seconds > 0
      ? Math.min(
          Math.max(Math.round(seconds), 60),
          IDLE_TIMEOUT_MAX_MINUTES * 60,
        )
      : 0;
  state.workspace.idleTimeoutSeconds = normalized;
  updateIdleTimeoutControls();
}

export function initializeWorkspaceSettingsUI() {
  initializeIdleTimeoutControls();
  updateIdleTimeoutControls();
}

function updateIdleTimeoutControls() {
  const toggle = elements.idleTimeoutToggle;
  const minutesInput = elements.idleTimeoutMinutes;
  if (!toggle || !minutesInput) {
    return;
  }
  const seconds = Number(state.workspace?.idleTimeoutSeconds || 0);
  const enabled = seconds > 0;
  toggle.checked = enabled;
  minutesInput.disabled =
    !enabled || state.workspace?.updatingIdleTimeout === true;
  if (enabled) {
    const minutes = Math.max(1, Math.round(seconds / 60));
    minutesInput.value = String(minutes);
  } else {
    minutesInput.value = "";
  }
  toggle.disabled = state.workspace?.updatingIdleTimeout === true;
}

function initializeIdleTimeoutControls() {
  if (state.workspace.idleControlsInitialized) {
    return;
  }
  const toggle = elements.idleTimeoutToggle;
  const minutesInput = elements.idleTimeoutMinutes;
  if (toggle) {
    toggle.addEventListener("change", handleIdleToggleChange);
  }
  if (minutesInput) {
    minutesInput.addEventListener("change", handleIdleMinutesChange);
    minutesInput.addEventListener("blur", handleIdleMinutesChange);
  }
  state.workspace.idleControlsInitialized = true;
  updateIdleTimeoutControls();
}

function handleIdleToggleChange(event) {
  const checked = event.currentTarget?.checked === true;
  const minutesInput = elements.idleTimeoutMinutes;
  if (!checked) {
    persistIdleTimeoutSeconds(0);
    return;
  }
  const currentMinutes = minutesInput?.value
    ? coerceMinutesValue(minutesInput.value)
    : null;
  const minutes = currentMinutes ?? IDLE_TIMEOUT_DEFAULT_MINUTES;
  if (minutesInput) {
    minutesInput.value = String(minutes);
  }
  persistIdleTimeoutSeconds(minutes * 60);
}

function handleIdleMinutesChange(event) {
  const toggle = elements.idleTimeoutToggle;
  if (!toggle || !toggle.checked) {
    return;
  }
  const minutes = coerceMinutesValue(event.currentTarget?.value);
  if (!minutes) {
    event.currentTarget.value = String(
      Math.max(1, Math.round((state.workspace.idleTimeoutSeconds || 60) / 60)),
    );
    return;
  }
  persistIdleTimeoutSeconds(minutes * 60);
}

function coerceMinutesValue(value) {
  const parsed = Number.parseInt(String(value), 10);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return null;
  }
  return Math.min(Math.max(parsed, 1), IDLE_TIMEOUT_MAX_MINUTES);
}

async function persistIdleTimeoutSeconds(seconds) {
  if (state.workspace.updatingIdleTimeout) {
    return;
  }
  const previous = state.workspace.idleTimeoutSeconds || 0;
  if (previous === seconds) {
    updateIdleTimeoutControls();
    return;
  }
  state.workspace.updatingIdleTimeout = true;
  state.workspace.idleTimeoutSeconds = seconds;
  updateIdleTimeoutControls();
  try {
    const response = await proxyToApi("/api/v1/workspace", {
      method: "PATCH",
      json: { idleTimeoutSeconds: seconds },
    });
    if (!response.ok) {
      const text = await response.text().catch(() => "");
      const { message } = parseApiError(text, "Unable to update idle timeout");
      throw new Error(message);
    }
  } catch (error) {
    console.error("Failed to update idle timeout:", error);
    state.workspace.idleTimeoutSeconds = previous;
    await showToast(
      error instanceof Error ? error.message : "Unable to update idle timeout",
      "error",
      3600,
    );
  } finally {
    state.workspace.updatingIdleTimeout = false;
    updateIdleTimeoutControls();
  }
}
