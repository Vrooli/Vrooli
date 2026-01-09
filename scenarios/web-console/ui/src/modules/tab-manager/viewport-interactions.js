// @ts-check

/** @typedef {import("../types.d.ts").TerminalTab} TerminalTab */

/**
 * Attach listeners that track when a user manually scrolls the xterm viewport.
 * Uses an AbortController to simplify teardown.
 * @param {TerminalTab} tab
 * @returns {() => void | null}
 */
export function attachViewportInteractionHandlers(tab) {
  if (typeof window === "undefined" || !tab?.container) {
    return null;
  }

  const viewport = tab.container.querySelector(".xterm-viewport");
  if (!(viewport instanceof HTMLElement)) {
    return null;
  }

  const controller = new AbortController();
  const { signal } = controller;

  const blockXtermTouch = (event) => {
    if (!(event instanceof TouchEvent)) {
      return;
    }
    if (typeof event.stopImmediatePropagation === "function") {
      event.stopImmediatePropagation();
    } else {
      event.stopPropagation();
    }
  };

  const propagationBlockTargets = new Set([
    tab.container,
    tab.container?.querySelector?.(".xterm"),
    tab.container?.querySelector?.(".xterm-screen"),
    tab.container?.querySelector?.(".xterm-helpers"),
    tab.container?.querySelector?.(".xterm-helper-textarea"),
    viewport,
  ]);

  propagationBlockTargets.forEach((target) => {
    if (!(target instanceof EventTarget)) {
      return;
    }
    target.addEventListener("touchstart", blockXtermTouch, {
      capture: true,
      passive: true,
      signal,
    });
    target.addEventListener("touchmove", blockXtermTouch, {
      capture: true,
      passive: false,
      signal,
    });
    target.addEventListener("touchend", blockXtermTouch, {
      capture: true,
      passive: true,
      signal,
    });
    target.addEventListener("touchcancel", blockXtermTouch, {
      capture: true,
      passive: true,
      signal,
    });
  });

  const clearReleaseTimer = () => {
    if (tab.userScroll.releaseTimer) {
      clearTimeout(tab.userScroll.releaseTimer);
      tab.userScroll.releaseTimer = null;
    }
  };

  const completeRelease = () => {
    clearReleaseTimer();
    tab.userScroll.active = false;
    tab.userScroll.momentumActive = false;
    tab.userScroll.lastInteraction = Date.now();
  };

  const scheduleRelease = (delay = 200) => {
    clearReleaseTimer();
    tab.userScroll.releaseTimer = window.setTimeout(() => {
      if (tab.userScroll.touchActive) {
        scheduleRelease(Math.max(220, delay));
        return;
      }
      completeRelease();
    }, Math.max(260, delay));
  };

  const updatePinnedState = () => {
    const atBottom =
      viewport.scrollHeight - viewport.clientHeight - viewport.scrollTop <= 2;

    if (tab.userScroll.touchActive) {
      tab.userScroll.momentumActive = true;
      if (!atBottom) {
        tab.userScroll.pinnedToBottom = false;
      }
      tab.userScroll.lastInteraction = Date.now();
      return;
    }

    tab.userScroll.pinnedToBottom = atBottom;
    if (atBottom) {
      completeRelease();
    } else {
      tab.userScroll.momentumActive = true;
      tab.userScroll.lastInteraction = Date.now();
    }
  };

  const markActive = () => {
    tab.userScroll.active = true;
    tab.userScroll.momentumActive = true;
    clearReleaseTimer();
    tab.userScroll.lastInteraction = Date.now();
  };

  const addListener = (target, type, handler, options = {}) => {
    if (!(target instanceof EventTarget)) {
      return;
    }
    const opts = typeof options === "object" && options !== null
      ? { ...options, signal }
      : { capture: Boolean(options), signal };
    target.addEventListener(type, handler, opts);
  };

  addListener(
    viewport,
    "scroll",
    () => {
      updatePinnedState();
      if (!tab.userScroll.pinnedToBottom) {
        markActive();
        scheduleRelease(700);
      }
    },
    { passive: true },
  );

  const pointerStart = () => {
    markActive();
    tab.userScroll.touchActive = true;
    tab.userScroll.pinnedToBottom = false;
    tab.userScroll.lastInteraction = Date.now();
  };

  const pointerEnd = () => {
    tab.userScroll.touchActive = false;
    tab.userScroll.momentumActive = true;
    tab.userScroll.lastInteraction = Date.now();
    scheduleRelease(900);
  };

  ["touchstart", "touchmove"].forEach((type) => {
    addListener(viewport, type, pointerStart, { passive: true });
  });
  ["touchend", "touchcancel"].forEach((type) => {
    addListener(viewport, type, pointerEnd, { passive: true });
  });

  addListener(
    viewport,
    "wheel",
    () => {
      markActive();
      scheduleRelease(640);
    },
    { passive: true },
  );

  const pointerTargets = new Set([
    viewport,
    tab.container,
    tab.container.querySelector(".xterm"),
    tab.container.querySelector(".xterm-screen"),
  ]);

  const pointerDown = (event) => {
    if (!(event instanceof PointerEvent) || !event.isPrimary) {
      return;
    }
    markActive();
    if (event.pointerType === "touch" || event.pointerType === "pen") {
      tab.userScroll.touchActive = true;
    }
  };

  const pointerMove = (event) => {
    if (!(event instanceof PointerEvent) || !event.isPrimary) {
      return;
    }
    if (event.pointerType === "touch" || event.pointerType === "pen") {
      tab.userScroll.active = true;
      tab.userScroll.lastInteraction = Date.now();
    }
  };

  const pointerUp = (event) => {
    if (!(event instanceof PointerEvent) || !event.isPrimary) {
      return;
    }
    if (event.pointerType === "touch" || event.pointerType === "pen") {
      tab.userScroll.touchActive = false;
      tab.userScroll.momentumActive = true;
      tab.userScroll.lastInteraction = Date.now();
      scheduleRelease(900);
    } else if (event.pointerType === "mouse") {
      tab.userScroll.lastInteraction = Date.now();
      scheduleRelease(520);
    }
  };

  pointerTargets.forEach((target) => {
    addListener(target, "pointerdown", pointerDown, {
      passive: true,
      capture: true,
    });
    addListener(target, "pointermove", pointerMove, {
      passive: true,
      capture: true,
    });
    addListener(target, "pointerup", pointerUp, {
      passive: true,
      capture: true,
    });
    addListener(target, "pointercancel", pointerUp, {
      passive: true,
      capture: true,
    });
  });

  const cleanup = () => {
    clearReleaseTimer();
    controller.abort();
    tab.userScroll.touchActive = false;
    tab.userScroll.momentumActive = false;
    tab.userScroll.active = false;
    tab.userScroll.pinnedToBottom = true;
    tab.userScroll.lastInteraction = Date.now();
    tab.userScroll.cleanup = null;
  };

  tab.userScroll.cleanup = cleanup;
  updatePinnedState();

  return cleanup;
}
