import { elements, state } from "./state.js";

let submitHandler = null;
let submitErrorHandler = null;
let getActiveTabFn = () => null;

export function configureComposer({ onSubmit, onSubmitError, getActiveTab }) {
  submitHandler = typeof onSubmit === "function" ? onSubmit : null;
  submitErrorHandler =
    typeof onSubmitError === "function" ? onSubmitError : null;
  getActiveTabFn =
    typeof getActiveTab === "function" ? getActiveTab : () => null;
}

export function initializeComposerUI() {
  if (elements.composeBtn) {
    elements.composeBtn.addEventListener("click", () => openComposeDialog());
  }
  if (elements.composeCancel) {
    elements.composeCancel.addEventListener("click", () =>
      closeComposeDialog({ preserveValue: false }),
    );
  }
  if (elements.composeClose) {
    elements.composeClose.addEventListener("click", () =>
      closeComposeDialog({ preserveValue: false }),
    );
  }
  if (elements.composeBackdrop) {
    elements.composeBackdrop.addEventListener("click", () =>
      closeComposeDialog({ preserveValue: false }),
    );
  }
  if (elements.composeForm) {
    elements.composeForm.addEventListener("submit", handleComposeSubmit);
  }
  if (elements.composeTextarea) {
    elements.composeTextarea.addEventListener("input", updateComposeFeedback);
  }
  if (elements.composeAppendNewline) {
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
  if (!elements.composeDialog || !elements.composeTextarea) return;
  if (state.composer.open) {
    elements.composeTextarea.focus();
    return;
  }

  if (!prefill && typeof window !== "undefined" && window.getSelection) {
    try {
      const selected = window.getSelection().toString();
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

  elements.composeDialog.classList.remove("hidden");
  elements.composeBackdrop?.classList.remove("hidden");
  elements.composeDialog.setAttribute("aria-hidden", "false");

  elements.composeTextarea.value = prefill || "";
  if (elements.composeAppendNewline) {
    elements.composeAppendNewline.checked =
      state.composer.appendNewline !== false;
  }
  updateComposeFeedback();

  requestAnimationFrame(() => {
    elements.composeTextarea?.focus();
    elements.composeTextarea?.select();
  });
}

export function closeComposeDialog(options = {}) {
  if (!state.composer.open) {
    return;
  }
  state.composer.open = false;

  elements.composeDialog?.classList.add("hidden");
  elements.composeBackdrop?.classList.add("hidden");
  elements.composeDialog?.setAttribute("aria-hidden", "true");

  if (!options.preserveValue && elements.composeTextarea) {
    elements.composeTextarea.value = "";
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
  if (!elements.composeTextarea || !elements.composeSend) return;
  const value = elements.composeTextarea.value || "";
  const newline = elements.composeAppendNewline
    ? elements.composeAppendNewline.checked
    : true;
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
  elements.composeSend.disabled = !tab || trimmed.length === 0;
}

function handleComposeSubmit(event) {
  event.preventDefault();
  if (!elements.composeTextarea) return;
  const tab = getActiveTabFn();
  if (!tab) {
    updateComposeFeedback();
    return;
  }

  const value = elements.composeTextarea.value;
  const trimmed = value.trim();
  if (!trimmed) {
    updateComposeFeedback();
    return;
  }

  const appendNewline = elements.composeAppendNewline
    ? elements.composeAppendNewline.checked
    : true;

  if (!submitHandler) {
    closeComposeDialog({ preserveValue: true });
    return;
  }

  const result = submitHandler({ tab, value, appendNewline });

  if (result && typeof result.then === "function") {
    result
      .then((sent) => handlePostSubmit(sent, value, appendNewline))
      .catch(() => handlePostSubmit(false, value, appendNewline));
  } else {
    handlePostSubmit(result !== false, value, appendNewline);
  }
}

function handlePostSubmit(success, value, appendNewline) {
  if (success) {
    elements.composeTextarea.value = "";
    updateComposeFeedback();
    closeComposeDialog({ preserveValue: false });
  } else {
    // Keep dialog open so the user can try again
    elements.composeTextarea.focus();
    if (submitErrorHandler) {
      submitErrorHandler({ value, appendNewline });
    }
  }
}
