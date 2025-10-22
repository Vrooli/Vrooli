import {
  elements,
  state,
  TAB_COLOR_DEFAULT,
  TAB_COLOR_OPTIONS,
  TAB_COLOR_MAP,
  TAB_LONG_PRESS_DELAY,
} from "../state.js";
import { tabCallbacks } from "./callbacks.js";
import {
  setActiveTab,
  findTab,
  applyTabAppearance,
  refreshTabButton,
} from "./tabs.js";

const tabColorButtons = new Map();
let tabMenuCloseTimer = null;

export function initializeTabCustomizationUI() {
  const container = elements.tabContextColors;
  if (!container) return;
  container.innerHTML = "";
  tabColorButtons.clear();

  TAB_COLOR_OPTIONS.forEach((option) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "tab-color-option";
    button.dataset.colorId = option.id;
    button.style.setProperty("--swatch-color", option.swatch);
    button.setAttribute("role", "option");
    button.setAttribute("aria-label", `${option.label} tab color`);
    button.addEventListener("click", () => {
      selectTabColor(option.id);
    });
    tabColorButtons.set(option.id, button);
    container.appendChild(button);
  });

  if (elements.tabContextForm) {
    elements.tabContextForm.addEventListener("submit", (event) => {
      event.preventDefault();
      handleTabContextSubmit();
    });
  }
  if (elements.tabContextCancel) {
    elements.tabContextCancel.addEventListener("click", () =>
      closeTabCustomization(),
    );
  }
  if (elements.tabContextBackdrop) {
    elements.tabContextBackdrop.addEventListener("click", () =>
      closeTabCustomization(),
    );
  }
  if (elements.tabContextReset) {
    elements.tabContextReset.addEventListener("click", handleTabContextReset);
  }

  selectTabColor(TAB_COLOR_DEFAULT);
}

export function registerTabCustomizationHandlers(button, tab) {
  if (!button) return;
  button.addEventListener("contextmenu", (event) => {
    event.preventDefault();
    handleTabCustomizationRequest(tab, event);
  });

  button.addEventListener("pointerdown", (event) => {
    if (event.pointerType === "mouse") return;
    let longPressTriggered = false;
    const timer = window.setTimeout(() => {
      longPressTriggered = true;
      handleTabCustomizationRequest(tab, event);
    }, TAB_LONG_PRESS_DELAY);

    const cancel = () => {
      window.clearTimeout(timer);
      button.removeEventListener("pointerup", cancel);
      button.removeEventListener("pointerleave", cancel);
      button.removeEventListener("pointercancel", cancel);
      if (longPressTriggered) {
        button.addEventListener("click", suppressNextClick, {
          once: true,
          capture: true,
        });
      }
    };

    button.addEventListener("pointerup", cancel);
    button.addEventListener("pointerleave", cancel);
    button.addEventListener("pointercancel", cancel);
  });
}

function handleTabCustomizationRequest(tab, triggerEvent) {
  if (!tab) return;
  let anchor;
  if (
    triggerEvent &&
    typeof triggerEvent.clientX === "number" &&
    typeof triggerEvent.clientY === "number"
  ) {
    anchor = { x: triggerEvent.clientX, y: triggerEvent.clientY };
  } else if (tab.domButton) {
    const rect = tab.domButton.getBoundingClientRect();
    anchor = { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 };
  } else {
    anchor = { x: window.innerWidth / 2, y: window.innerHeight / 2 };
  }
  setActiveTab(tab.id);
  openTabCustomization(tab, anchor);
  tabCallbacks.onTabCustomizationOpened?.(tab);
}

function handleTabContextSubmit() {
  const tabId = state.tabMenu.tabId;
  const tab = tabId ? findTab(tabId) : null;
  if (!tab) {
    closeTabCustomization();
    return;
  }

  const inputValue = elements.tabContextName?.value?.trim() || "";
  const colorId =
    state.tabMenu.selectedColor && TAB_COLOR_MAP[state.tabMenu.selectedColor]
      ? state.tabMenu.selectedColor
      : TAB_COLOR_DEFAULT;

  tab.label = inputValue || tab.defaultLabel || tab.label;
  tab.colorId = colorId;

  if (tab.domLabel) {
    tab.domLabel.textContent = tab.label;
  }
  if (tab.domButton) {
    tab.domButton.title = tab.label;
  }
  if (tab.domClose) {
    tab.domClose.setAttribute("aria-label", `Close ${tab.label}`);
  }

  applyTabAppearance(tab);
  refreshTabButton(tab);

  tabCallbacks.onTabMetadataChanged?.(tab);

  closeTabCustomization();
}

function handleTabContextReset() {
  if (!state.tabMenu.open) return;
  const tabId = state.tabMenu.tabId;
  const tab = tabId ? findTab(tabId) : null;
  if (!tab) {
    closeTabCustomization();
    return;
  }
  const fallbackLabel = tab.defaultLabel || tab.label;
  if (elements.tabContextName) {
    elements.tabContextName.value = fallbackLabel;
  }
  state.tabMenu.selectedColor = TAB_COLOR_DEFAULT;
  selectTabColor(TAB_COLOR_DEFAULT);
  tabCallbacks.onTabMetadataChanged?.(tab);
}

function suppressNextClick(event) {
  event.preventDefault();
  event.stopImmediatePropagation();
}

export function openTabCustomization(tab, anchor) {
  if (
    !elements.tabContextMenu ||
    !elements.tabContextBackdrop ||
    !elements.tabContextName
  )
    return;

  if (tabMenuCloseTimer) {
    window.clearTimeout(tabMenuCloseTimer);
    tabMenuCloseTimer = null;
  }

  const option = TAB_COLOR_MAP[tab.colorId] ? tab.colorId : TAB_COLOR_DEFAULT;

  state.tabMenu.open = true;
  state.tabMenu.tabId = tab.id;
  state.tabMenu.selectedColor = option;
  state.tabMenu.anchor = anchor || {
    x: window.innerWidth / 2,
    y: window.innerHeight / 2,
  };

  elements.tabContextName.value = tab.label;
  selectTabColor(state.tabMenu.selectedColor);

  elements.tabContextBackdrop.classList.remove("hidden");
  elements.tabContextMenu.classList.remove("hidden");
  elements.tabContextMenu.style.left = "0px";
  elements.tabContextMenu.style.top = "0px";

  requestAnimationFrame(() => {
    elements.tabContextBackdrop.classList.add("active");
    elements.tabContextBackdrop.setAttribute("aria-hidden", "false");
    positionTabContextMenu(state.tabMenu.anchor);
    elements.tabContextMenu.classList.add("active");
    elements.tabContextMenu.setAttribute("aria-hidden", "false");
    try {
      elements.tabContextName.focus({ preventScroll: true });
    } catch (_error) {
      elements.tabContextName.focus();
    }
    elements.tabContextName.select();
  });
}

export function closeTabCustomization() {
  if (!state.tabMenu.open) return;
  state.tabMenu.open = false;
  state.tabMenu.tabId = null;
  state.tabMenu.anchor = { x: 0, y: 0 };
  state.tabMenu.selectedColor = TAB_COLOR_DEFAULT;

  if (!elements.tabContextMenu || !elements.tabContextBackdrop) return;

  elements.tabContextMenu.classList.remove("active");
  elements.tabContextMenu.setAttribute("aria-hidden", "true");
  elements.tabContextBackdrop.classList.remove("active");
  elements.tabContextBackdrop.setAttribute("aria-hidden", "true");

  tabMenuCloseTimer = window.setTimeout(() => {
    elements.tabContextMenu?.classList.add("hidden");
    elements.tabContextBackdrop?.classList.add("hidden");
    tabMenuCloseTimer = null;
    tabCallbacks.onTabCustomizationClosed?.();
  }, 180);
}

export function positionTabContextMenu(anchor) {
  if (!elements.tabContextMenu) return;
  const menu = elements.tabContextMenu;
  const { x, y } = anchor || {
    x: window.innerWidth / 2,
    y: window.innerHeight / 2,
  };

  const padding = 16;
  const rect = menu.getBoundingClientRect();

  let left = x;
  let top = y;

  if (left + rect.width + padding > window.innerWidth) {
    left = window.innerWidth - rect.width - padding;
  }
  if (left < padding) {
    left = padding;
  }
  if (top + rect.height + padding > window.innerHeight) {
    top = window.innerHeight - rect.height - padding;
  }
  if (top < padding) {
    top = padding;
  }

  menu.style.left = `${left}px`;
  menu.style.top = `${top}px`;
}

function selectTabColor(colorId) {
  const option = TAB_COLOR_MAP[colorId] ? colorId : TAB_COLOR_DEFAULT;
  state.tabMenu.selectedColor = option;
  tabColorButtons.forEach((button, id) => {
    const isSelected = id === option;
    button.setAttribute("aria-selected", isSelected ? "true" : "false");
    button.classList.toggle("selected", isSelected);
  });
}
