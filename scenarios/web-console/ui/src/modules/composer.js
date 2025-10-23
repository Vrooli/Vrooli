// @ts-check

import { elements, state } from "./state.js";

/** @typedef {import("./types.d.ts").TerminalTab} TerminalTab */

/**
 * @typedef {{ tab: TerminalTab; value: string; appendNewline: boolean }} ComposerSubmitPayload
 * @typedef {{ value: string; appendNewline: boolean }} ComposerErrorPayload
 */

/** @type {((payload: ComposerSubmitPayload) => boolean | void | Promise<boolean | void>) | null} */
let submitHandler = null;
/** @type {((payload: ComposerErrorPayload) => void) | null} */
let submitErrorHandler = null;
/** @type {() => TerminalTab | null} */
let getActiveTabFn = () => null;

/**
 * @param {{
 *  onSubmit?: (payload: ComposerSubmitPayload) => boolean | void | Promise<boolean | void>;
 *  onSubmitError?: (payload: ComposerErrorPayload) => void;
 *  getActiveTab?: () => TerminalTab | null;
 * }} [options]
 */
export function configureComposer({ onSubmit, onSubmitError, getActiveTab } = {}) {
  submitHandler = typeof onSubmit === "function" ? onSubmit : null;
  submitErrorHandler =
    typeof onSubmitError === "function" ? onSubmitError : null;
  getActiveTabFn =
    typeof getActiveTab === "function" ? getActiveTab : () => null;
}

export function initializeComposerUI() {
  if (elements.composeBtn instanceof HTMLButtonElement) {
    elements.composeBtn.addEventListener("click", () => openComposeDialog());
  }
  if (elements.composeCancel instanceof HTMLButtonElement) {
    elements.composeCancel.addEventListener("click", () =>
      closeComposeDialog({ preserveValue: false }),
    );
  }
  if (elements.composeClose instanceof HTMLButtonElement) {
    elements.composeClose.addEventListener("click", () =>
      closeComposeDialog({ preserveValue: false }),
    );
  }
  if (elements.composeBackdrop) {
    elements.composeBackdrop.addEventListener("click", () =>
      closeComposeDialog({ preserveValue: false }),
    );
  }
  if (elements.composeForm instanceof HTMLFormElement) {
    elements.composeForm.addEventListener("submit", handleComposeSubmit);
  }
  if (elements.composeTextarea instanceof HTMLTextAreaElement) {
    elements.composeTextarea.addEventListener("input", updateComposeFeedback);
  }
  if (elements.composeAppendNewline instanceof HTMLInputElement) {
    elements.composeAppendNewline.addEventListener(
      "change",
      updateComposeFeedback,
    );
  }
  updateComposeFeedback();
}

export function isComposerOpen() {
  return state.composer.open === true;
}

export function openComposeDialog(prefill = "") {
  const dialog = elements.composeDialog;
  const textarea =
    elements.composeTextarea instanceof HTMLTextAreaElement
      ? elements.composeTextarea
      : null;
  const appendToggle =
    elements.composeAppendNewline instanceof HTMLInputElement
      ? elements.composeAppendNewline
      : null;
  if (!dialog || !textarea) return;
  if (state.composer.open) {
    textarea.focus();
    return;
  }

  if (!prefill && typeof window !== "undefined" && window.getSelection) {
    try {
      const selection = window.getSelection();
      const selected = selection ? selection.toString() : "";
      if (selected && selected.trim().length > 0) {
        prefill = selected.trimEnd();
      }
    } catch (_error) {
      // ignore selection access issues
    }
  }

  state.composer.open = true;
  state.composer.previousFocus =
    document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null;

  dialog.classList.remove("hidden");
  elements.composeBackdrop?.classList.remove("hidden");
  dialog.setAttribute("aria-hidden", "false");

  textarea.value = prefill || "";
  if (appendToggle) {
    appendToggle.checked = state.composer.appendNewline !== false;
  }
  updateComposeFeedback();

  requestAnimationFrame(() => {
    textarea.focus();
    textarea.select();
  });
}

/**
 * @param {{ preserveValue?: boolean }} [options]
 */
export function closeComposeDialog(options = {}) {
  if (!state.composer.open) {
    return;
  }
  state.composer.open = false;

  const dialog = elements.composeDialog;
  const textarea =
    elements.composeTextarea instanceof HTMLTextAreaElement
      ? elements.composeTextarea
      : null;
  dialog?.classList.add("hidden");
  elements.composeBackdrop?.classList.add("hidden");
  dialog?.setAttribute("aria-hidden", "true");

  const preserveValue = options.preserveValue === true;
  if (!preserveValue && textarea) {
    textarea.value = "";
  }

  updateComposeFeedback();

  if (
    state.composer.previousFocus &&
    typeof state.composer.previousFocus.focus === "function"
  ) {
    try {
      state.composer.previousFocus.focus();
    } catch (_error) {
      // ignore focus failures
    }
  }
  state.composer.previousFocus = null;
}

export function updateComposeFeedback() {
  const textarea =
    elements.composeTextarea instanceof HTMLTextAreaElement
      ? elements.composeTextarea
      : null;
  const sendButton =
    elements.composeSend instanceof HTMLButtonElement
      ? elements.composeSend
      : null;
  const appendToggle =
    elements.composeAppendNewline instanceof HTMLInputElement
      ? elements.composeAppendNewline
      : null;
  if (!textarea || !sendButton) return;
  const value = textarea.value || "";
  const newline = appendToggle ? appendToggle.checked : true;
  state.composer.appendNewline = newline;

  const needsNewline = newline && !value.endsWith("\n");

  const tab = getActiveTabFn();

  if (elements.composeCharCount) {
    const totalChars = value.length + (needsNewline ? 1 : 0);
    const baseLabel = needsNewline
      ? `${totalChars} chars (includes appended newline)`
      : `${totalChars} chars`;
    elements.composeCharCount.textContent = tab
      ? baseLabel
      : `No active terminal · ${baseLabel}`;
  }

  const trimmed = value.trim();
  sendButton.disabled = !tab || trimmed.length === 0;
}

/**
 * @param {SubmitEvent} event
 */
function handleComposeSubmit(event) {
  event.preventDefault();
  const textarea =
    elements.composeTextarea instanceof HTMLTextAreaElement
      ? elements.composeTextarea
      : null;
  if (!textarea) return;
  const tab = getActiveTabFn();
  if (!tab) {
    updateComposeFeedback();
    return;
  }

  const value = textarea.value;
  const trimmed = value.trim();
  if (!trimmed) {
    updateComposeFeedback();
    return;
  }

  const appendToggle =
    elements.composeAppendNewline instanceof HTMLInputElement
      ? elements.composeAppendNewline
      : null;
  const appendNewline = appendToggle ? appendToggle.checked : true;

  if (!submitHandler) {
    closeComposeDialog({ preserveValue: true });
    return;
  }

  const outcome = submitHandler({ tab, value, appendNewline });

  Promise.resolve(outcome)
    .then((sent) => handlePostSubmit(sent !== false, value, appendNewline))
    .catch(() => handlePostSubmit(false, value, appendNewline));
}

/**
 * @param {boolean} success
 * @param {string} value
 * @param {boolean} appendNewline
 */
function handlePostSubmit(success, value, appendNewline) {
  if (success) {
    const textarea =
      elements.composeTextarea instanceof HTMLTextAreaElement
        ? elements.composeTextarea
        : null;
    if (textarea) {
      textarea.value = "";
    }
    updateComposeFeedback();
    closeComposeDialog({ preserveValue: false });
  } else {
    // Keep dialog open so the user can try again
    const textarea =
      elements.composeTextarea instanceof HTMLTextAreaElement
        ? elements.composeTextarea
        : null;
    textarea?.focus();
    if (submitErrorHandler) {
      submitErrorHandler({ value, appendNewline });
    }
  }
}
