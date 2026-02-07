export type ShortcutState = {
  keys: string[];
  description: string;
};

export const TAB_SWITCHER_SHORTCUT_KEY = 'k';

const isEditableTarget = (target: EventTarget | null): target is HTMLElement => {
  if (!(target instanceof HTMLElement)) {
    return false;
  }
  return target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable;
};

export const isTabSwitcherShortcutEvent = (event: KeyboardEvent): boolean => {
  const key = event.key?.toLowerCase();
  if (key !== TAB_SWITCHER_SHORTCUT_KEY) {
    return false;
  }

  if (!(event.ctrlKey || event.metaKey) || event.altKey) {
    return false;
  }

  if (isEditableTarget(event.target)) {
    return false;
  }

  return true;
};

export const resolveTabSwitcherShortcut = (
  nav: (Navigator & { userAgentData?: { platform?: string } }) | undefined = typeof navigator !== 'undefined' ? navigator as Navigator & { userAgentData?: { platform?: string } } : undefined,
): ShortcutState | null => {
  if (!nav) {
    return null;
  }

  const platform = `${nav.platform ?? ''} ${nav.userAgentData?.platform ?? ''}`.toLowerCase();
  const userAgent = (nav.userAgent ?? '').toLowerCase();
  const combined = `${platform} ${userAgent}`;
  const maxTouchPoints = typeof nav.maxTouchPoints === 'number' ? nav.maxTouchPoints : 0;

  const isIOS = /iphone|ipad|ipod/.test(combined)
    || (/mac/.test(platform) && maxTouchPoints > 1 && /ipad|iphone/.test(userAgent));
  const isAndroid = /android/.test(combined);
  const isMobile = isIOS || isAndroid || /mobile/.test(combined);

  if (isMobile) {
    return null;
  }

  const isMac = /mac/.test(combined) && !isIOS;

  if (isMac) {
    return {
      keys: ['⌘', 'K'],
      description: 'Command plus K',
    };
  }

  return {
    keys: ['Ctrl', 'K'],
    description: 'Control plus K',
  };
};
