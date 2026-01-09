import { elements, state } from "./state.js";

const FLOATING_DRAWER_QUERY = "(max-width: 1024px)";

export function isDrawerFloating() {
  if (
    typeof window === "undefined" ||
    typeof window.matchMedia !== "function"
  ) {
    return false;
  }
  try {
    return window.matchMedia(FLOATING_DRAWER_QUERY).matches;
  } catch (_error) {
    return false;
  }
}

export function applyDrawerState() {
  if (!elements.layout || !elements.detailsDrawer || !elements.drawerToggle)
    return;
  elements.layout.classList.toggle("drawer-open", state.drawer.open);
  elements.detailsDrawer.setAttribute(
    "aria-hidden",
    state.drawer.open ? "false" : "true",
  );
  if (state.drawer.open) {
    elements.detailsDrawer.removeAttribute("inert");
  } else {
    elements.detailsDrawer.setAttribute("inert", "");
  }
  elements.drawerToggle.setAttribute(
    "aria-expanded",
    state.drawer.open ? "true" : "false",
  );
  if (elements.drawerBackdrop) {
    elements.drawerBackdrop.setAttribute(
      "aria-hidden",
      state.drawer.open ? "false" : "true",
    );
  }
}

export function updateDrawerIndicator() {
  if (!elements.drawerIndicator) return;
  const count = state.drawer?.unreadCount || 0;
  const hasUnread = count > 0;
  if (elements.drawerToggle) {
    elements.drawerToggle.setAttribute(
      "aria-label",
      hasUnread
        ? "Open session details (new activity)"
        : "Open session details",
    );
    elements.drawerToggle.classList.toggle("has-unread", hasUnread);
  }
  if (hasUnread) {
    elements.drawerIndicator.classList.remove("hidden");
    elements.drawerIndicator.setAttribute("aria-hidden", "false");
  } else {
    elements.drawerIndicator.classList.add("hidden");
    elements.drawerIndicator.setAttribute("aria-hidden", "true");
  }
}

export function resetUnreadEvents() {
  if (!state.drawer) return;
  if (state.drawer.unreadCount !== 0) {
    state.drawer.unreadCount = 0;
  }
  updateDrawerIndicator();
}

export function openDrawer() {
  if (state.drawer.open) return;
  state.drawer.open = true;
  state.drawer.previousFocus =
    document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null;
  applyDrawerState();
  resetUnreadEvents();
  requestAnimationFrame(() => {
    if (
      !elements.detailsDrawer ||
      typeof elements.detailsDrawer.focus !== "function"
    )
      return;
    try {
      elements.detailsDrawer.focus({ preventScroll: true });
    } catch (_error) {
      elements.detailsDrawer.focus();
    }
  });
}

export function closeDrawer() {
  if (!state.drawer.open) return;
  state.drawer.open = false;
  applyDrawerState();
  updateDrawerIndicator();
  const fallback =
    state.drawer.previousFocus && state.drawer.previousFocus.isConnected
      ? state.drawer.previousFocus
      : elements.drawerToggle;
  requestAnimationFrame(() => {
    fallback?.focus?.();
    state.drawer.previousFocus = null;
  });
}

/**
 * @param {boolean} [forceState]
 */
export function toggleDrawer(forceState) {
  if (typeof forceState === "boolean") {
    return forceState ? openDrawer() : closeDrawer();
  }
  if (state.drawer.open) {
    closeDrawer();
  } else {
    openDrawer();
  }
}
