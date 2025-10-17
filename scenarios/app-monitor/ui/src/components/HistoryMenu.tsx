import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type {
  ChangeEvent,
  CSSProperties,
  KeyboardEvent as ReactKeyboardEvent,
} from 'react';
import { createPortal } from 'react-dom';
import clsx from 'clsx';
import { History } from 'lucide-react';
import type { App } from '@/types';
import { parseTimestampValue, resolveAppIdentifier } from '@/utils/appPreview';

const HISTORY_MENU_OFFSET = 10;
const HISTORY_MENU_MARGIN = 12;

const HIDDEN_MENU_STYLE: CSSProperties = {
  top: '-9999px',
  left: '-9999px',
  visibility: 'hidden',
};

interface MenuItem {
  app: App;
  key: string;
}

const formatRelativeViewed = (value?: string | null): string | null => {
  const timestamp = parseTimestampValue(value);
  if (timestamp === null) {
    return null;
  }

  const diffMs = Date.now() - timestamp;
  const diffSeconds = Math.floor(diffMs / 1000);

  if (diffSeconds < 30) {
    return 'Just now';
  }
  if (diffSeconds < 90) {
    return '1 minute ago';
  }

  const diffMinutes = Math.floor(diffSeconds / 60);
  if (diffMinutes < 60) {
    return `${diffMinutes} minute${diffMinutes === 1 ? '' : 's'} ago`;
  }

  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) {
    return `${diffHours} hour${diffHours === 1 ? '' : 's'} ago`;
  }

  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 7) {
    return `${diffDays} day${diffDays === 1 ? '' : 's'} ago`;
  }

  const diffWeeks = Math.floor(diffDays / 7);
  if (diffWeeks < 5) {
    return `${diffWeeks} week${diffWeeks === 1 ? '' : 's'} ago`;
  }

  const diffMonths = Math.floor(diffDays / 30);
  if (diffMonths < 12) {
    return `${diffMonths} month${diffMonths === 1 ? '' : 's'} ago`;
  }

  const diffYears = Math.floor(diffDays / 365);
  return `${diffYears} year${diffYears === 1 ? '' : 's'} ago`;
};

const formatViewCount = (value?: number): string | null => {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return null;
  }

  const count = Math.max(0, Math.floor(value));
  if (count === 0) {
    return null;
  }

  return `${count} view${count === 1 ? '' : 's'}`;
};

const getAppLabel = (app: App): string => {
  const candidates: Array<string | undefined> = [
    app.scenario_name,
    app.name,
    app.id,
    resolveAppIdentifier(app) ?? undefined,
  ];

  const label = candidates.find(value => typeof value === 'string' && value.trim().length > 0);
  return label ? label.trim() : 'Unknown app';
};

const matchesQuery = (app: App, normalizedQuery: string): boolean => {
  if (!normalizedQuery) {
    return true;
  }

  return getAppLabel(app).toLowerCase().includes(normalizedQuery);
};

interface HistoryMenuProps {
  recentApps: App[];
  allApps: App[];
  shouldShowHistory: boolean;
  onSelect(app: App): void;
  containerClassName?: string;
  buttonClassName?: string;
  iconClassName?: string;
  portalContainer?: HTMLElement | null;
  onToggle?: (isOpen: boolean) => void;
  usePortal?: boolean;
  closeSignal?: number;
}

export default function HistoryMenu({
  recentApps,
  allApps,
  shouldShowHistory,
  onSelect,
  containerClassName,
  buttonClassName,
  iconClassName,
  portalContainer,
  onToggle,
  usePortal = true,
  closeSignal,
}: HistoryMenuProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const [menuStyle, setMenuStyle] = useState<CSSProperties>(HIDDEN_MENU_STYLE);
  const [searchQuery, setSearchQuery] = useState('');
  const buttonRef = useRef<HTMLButtonElement | null>(null);
  const menuRef = useRef<HTMLDivElement | null>(null);
  const searchInputRef = useRef<HTMLInputElement | null>(null);
  const itemRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const fallbackKeyMap = useRef(new WeakMap<App, string>());
  const fallbackKeyCounter = useRef(0);
  const isBrowser = typeof document !== 'undefined';
  const portalHost = portalContainer ?? (isBrowser ? document.body : null);
  const closeSignalRef = useRef(closeSignal);

  const normalizedQuery = searchQuery.trim().toLowerCase();

  const closeMenu = useCallback(() => {
    setIsOpen(false);
    setMenuStyle(HIDDEN_MENU_STYLE);
    setActiveIndex(null);
    setSearchQuery('');
    onToggle?.(false);
  }, [onToggle]);

  const openMenu = useCallback(() => {
    setIsOpen(true);
    onToggle?.(true);
  }, [onToggle]);

  const getAppKey = useCallback((app: App): string => {
    const candidates: Array<string | undefined> = [
      app.id,
      app.scenario_name,
      app.name,
      resolveAppIdentifier(app) ?? undefined,
    ];
    const resolved = candidates.find(value => typeof value === 'string' && value.trim().length > 0);
    if (resolved) {
      return resolved.trim().toLowerCase();
    }

    const existing = fallbackKeyMap.current.get(app);
    if (existing) {
      return existing;
    }

    const generated = `app-${fallbackKeyCounter.current++}`;
    fallbackKeyMap.current.set(app, generated);
    return generated;
  }, []);

  const filteredRecentApps = useMemo(() => {
    if (recentApps.length === 0) {
      return [] as App[];
    }

    if (!normalizedQuery) {
      return recentApps;
    }

    return recentApps.filter(app => matchesQuery(app, normalizedQuery));
  }, [normalizedQuery, recentApps]);

  const filteredAllApps = useMemo(() => {
    if (allApps.length === 0) {
      return [] as App[];
    }

    const recentKeys = new Set(
      filteredRecentApps
        .map(app => getAppKey(app))
        .filter((key): key is string => Boolean(key)),
    );

    return allApps.filter(app => {
      const key = getAppKey(app);
      if (recentKeys.has(key)) {
        return false;
      }

      return matchesQuery(app, normalizedQuery);
    });
  }, [allApps, filteredRecentApps, getAppKey, normalizedQuery]);

  const menuItems = useMemo(() => {
    const recentItems: MenuItem[] = filteredRecentApps.map(app => ({
      app,
      key: `recent-${getAppKey(app)}`,
    }));

    const allItems: MenuItem[] = filteredAllApps.map(app => ({
      app,
      key: `all-${getAppKey(app)}`,
    }));

    return {
      recentItems,
      allItems,
      flatItems: [...recentItems, ...allItems],
    } as const;
  }, [filteredAllApps, filteredRecentApps, getAppKey]);

  const { recentItems, allItems, flatItems } = menuItems;
  const hasRecentSection = recentItems.length > 0;
  const hasAllSection = allItems.length > 0;
  const totalItems = flatItems.length;

  const updateMenuStyle = useCallback(() => {
    if (!isBrowser) {
      return;
    }

    const button = buttonRef.current;
    if (!button) {
      setMenuStyle(HIDDEN_MENU_STYLE);
      return;
    }

    const anchorRect = button.getBoundingClientRect();
    const menu = menuRef.current;
    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;
    const menuRect = menu?.getBoundingClientRect();
    const menuHeight = menuRect?.height ?? 0;
    const menuWidth = menuRect?.width ?? 0;

    let top = anchorRect.bottom + HISTORY_MENU_OFFSET;
    if (Number.isFinite(viewportHeight)) {
      const spaceBelow = viewportHeight - anchorRect.bottom - HISTORY_MENU_OFFSET;
      const shouldPlaceBelow = menuHeight === 0
        ? anchorRect.top + (anchorRect.height / 2) < viewportHeight / 2
        : spaceBelow >= menuHeight + HISTORY_MENU_MARGIN;
      if (!shouldPlaceBelow) {
        top = anchorRect.top - HISTORY_MENU_OFFSET - menuHeight;
      }
      const maxTop = viewportHeight - HISTORY_MENU_MARGIN - menuHeight;
      const minTop = HISTORY_MENU_MARGIN;
      top = Math.min(Math.max(top, minTop), maxTop);
    }

    let left = anchorRect.right;
    if (Number.isFinite(viewportWidth)) {
      const maxLeft = viewportWidth - HISTORY_MENU_MARGIN;
      const minLeft = menuWidth > 0 ? menuWidth + HISTORY_MENU_MARGIN : HISTORY_MENU_MARGIN;
      left = Math.min(Math.max(left, minLeft), maxLeft);
    }

    setMenuStyle({
      top: `${Math.round(top)}px`,
      left: `${Math.round(left)}px`,
      transform: 'translateX(-100%)',
      visibility: 'visible',
    });
  }, [isBrowser]);

  useEffect(() => {
    if (!shouldShowHistory && isOpen) {
      closeMenu();
    }
  }, [closeMenu, isOpen, shouldShowHistory]);

  useEffect(() => {
    if (typeof closeSignal !== 'number') {
      return;
    }

    if (closeSignalRef.current === closeSignal) {
      return;
    }

    closeSignalRef.current = closeSignal;

    if (isOpen) {
      closeMenu();
    }
  }, [closeSignal, closeMenu, isOpen]);

  useEffect(() => {
    if (!isOpen) {
      setMenuStyle(HIDDEN_MENU_STYLE);
      itemRefs.current = [];
      return;
    }

    let animationFrame = 0;

    const scheduleUpdate = () => {
      if (typeof window === 'undefined') {
        return;
      }
      animationFrame = window.requestAnimationFrame(() => {
        updateMenuStyle();
      });
    };

    scheduleUpdate();

    const handleResizeOrScroll = () => {
      updateMenuStyle();
    };

    window.addEventListener('resize', handleResizeOrScroll);
    window.addEventListener('scroll', handleResizeOrScroll, true);

    return () => {
      if (typeof window !== 'undefined') {
        window.cancelAnimationFrame(animationFrame);
        window.removeEventListener('resize', handleResizeOrScroll);
        window.removeEventListener('scroll', handleResizeOrScroll, true);
      }
    };
  }, [isOpen, totalItems, hasRecentSection, hasAllSection, updateMenuStyle]);

  useEffect(() => {
    if (isOpen) {
      updateMenuStyle();
    }
  }, [isOpen, normalizedQuery, updateMenuStyle]);

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    const handlePointer = (event: MouseEvent | TouchEvent) => {
      const target = event.target as Node | null;
      if (!target) {
        return;
      }

      if (buttonRef.current?.contains(target)) {
        return;
      }

      if (menuRef.current?.contains(target)) {
        return;
      }

      closeMenu();
      buttonRef.current?.focus();
    };

    const handleKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        closeMenu();
        buttonRef.current?.focus();
      }
    };

    document.addEventListener('mousedown', handlePointer);
    document.addEventListener('touchstart', handlePointer, { passive: true });
    document.addEventListener('keydown', handleKey);

    return () => {
      document.removeEventListener('mousedown', handlePointer);
      document.removeEventListener('touchstart', handlePointer);
      document.removeEventListener('keydown', handleKey);
    };
  }, [closeMenu, isOpen]);

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    itemRefs.current = itemRefs.current.slice(0, totalItems);

    const frame = window.requestAnimationFrame(() => {
      if (activeIndex === null) {
        searchInputRef.current?.focus();
        return;
      }

      const normalizedIndex = Math.min(Math.max(activeIndex, 0), Math.max(totalItems - 1, 0));
      const target = itemRefs.current[normalizedIndex];
      if (target) {
        target.focus();
        if (normalizedIndex !== activeIndex) {
          setActiveIndex(normalizedIndex);
        }
      } else if (totalItems === 0) {
        setActiveIndex(null);
        searchInputRef.current?.focus();
      }
    });

    return () => {
      window.cancelAnimationFrame(frame);
    };
  }, [activeIndex, isOpen, totalItems]);

  useEffect(() => {
    if (activeIndex !== null && (activeIndex < 0 || activeIndex >= totalItems)) {
      setActiveIndex(totalItems > 0 ? Math.min(activeIndex, totalItems - 1) : null);
    }
  }, [activeIndex, totalItems]);

  const focusItem = useCallback((index: number) => {
    if (totalItems === 0) {
      return;
    }

    const normalizedIndex = ((index % totalItems) + totalItems) % totalItems;
    const target = itemRefs.current[normalizedIndex];
    if (target) {
      target.focus();
      setActiveIndex(normalizedIndex);
    }
  }, [totalItems]);

  const handleToggle = useCallback(() => {
    if (!shouldShowHistory) {
      return;
    }

    if (isOpen) {
      closeMenu();
      buttonRef.current?.focus();
    } else {
      openMenu();
    }
  }, [buttonRef, closeMenu, isOpen, openMenu, shouldShowHistory]);

  const handleToggleKeyDown = useCallback((event: ReactKeyboardEvent<HTMLButtonElement>) => {
    if (!shouldShowHistory) {
      return;
    }

    if (event.key === 'ArrowDown') {
      event.preventDefault();
      setActiveIndex(totalItems > 0 ? 0 : null);
      if (!isOpen) {
        openMenu();
      }
      return;
    }

    if (event.key === 'ArrowUp') {
      event.preventDefault();
      setActiveIndex(totalItems > 0 ? totalItems - 1 : null);
      if (!isOpen) {
        openMenu();
      }
      return;
    }

    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      if (isOpen) {
        closeMenu();
        buttonRef.current?.focus();
      } else {
        openMenu();
      }
    }
  }, [buttonRef, closeMenu, isOpen, openMenu, shouldShowHistory, totalItems]);

  const handleSelect = useCallback((app: App) => {
    closeMenu();
    onSelect(app);
  }, [closeMenu, onSelect]);

  const handleItemKeyDown = useCallback((index: number) => (event: ReactKeyboardEvent<HTMLButtonElement>) => {
    if (event.key === 'ArrowDown') {
      event.preventDefault();
      focusItem(index + 1);
      return;
    }

    if (event.key === 'ArrowUp') {
      event.preventDefault();
      if (index === 0) {
        searchInputRef.current?.focus();
        setActiveIndex(null);
      } else {
        focusItem(index - 1);
      }
      return;
    }

    if (event.key === 'Home') {
      event.preventDefault();
      focusItem(0);
      return;
    }

    if (event.key === 'End') {
      event.preventDefault();
      focusItem(totalItems - 1);
      return;
    }

    if (event.key === 'Escape') {
      event.preventDefault();
      closeMenu();
      buttonRef.current?.focus();
    }
  }, [buttonRef, closeMenu, focusItem, totalItems]);

  const handleSearchChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    setSearchQuery(event.target.value);
  }, []);

  const handleSearchKeyDown = useCallback((event: ReactKeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'ArrowDown') {
      event.preventDefault();
      if (totalItems > 0) {
        focusItem(0);
      }
      return;
    }

    if (event.key === 'ArrowUp') {
      event.preventDefault();
      if (totalItems > 0) {
        focusItem(totalItems - 1);
      } else {
        buttonRef.current?.focus();
        setActiveIndex(null);
      }
      return;
    }

    if (event.key === 'Escape') {
      event.preventDefault();
      closeMenu();
      buttonRef.current?.focus();
    }
  }, [buttonRef, closeMenu, focusItem, totalItems]);

  if (!shouldShowHistory) {
    return null;
  }

  const panel = (
    <div
      className="history-menu__panel"
      role="menu"
      aria-label="App quick switcher"
      ref={menuRef}
      style={menuStyle}
    >
      <div className="history-menu__search">
        <input
          ref={searchInputRef}
          className="history-menu__search-input"
          type="search"
          placeholder="Search apps"
          value={searchQuery}
          onChange={handleSearchChange}
          onKeyDown={handleSearchKeyDown}
          aria-label="Search apps"
        />
      </div>
      <div className="history-menu__sections">
        {hasRecentSection && (
          <div className="history-menu__section" role="group" aria-label="Recently viewed">
            <div className="history-menu__header">Recently viewed</div>
            <ul className="history-menu__list" role="none">
              {recentItems.map((item, indexInSection) => {
                const flatIndex = indexInSection;
                const { app } = item;
                const label = getAppLabel(app);
                const relativeViewed = formatRelativeViewed(app.last_viewed_at);
                const viewCountRaw = Number(app.view_count ?? 0);
                const viewCountLabel = Number.isFinite(viewCountRaw)
                  ? formatViewCount(viewCountRaw)
                  : null;

                return (
                  <li key={item.key} role="none">
                    <button
                      type="button"
                      role="menuitem"
                      className={clsx('history-menu__item', {
                        'is-active': activeIndex === flatIndex,
                      })}
                      ref={(element) => {
                        itemRefs.current[flatIndex] = element ?? null;
                      }}
                      onClick={() => handleSelect(app)}
                      onKeyDown={handleItemKeyDown(flatIndex)}
                      aria-current={activeIndex === flatIndex ? 'true' : undefined}
                    >
                      <span className="history-menu__item-name">{label}</span>
                      <span className="history-menu__item-meta">
                        {relativeViewed && (
                          <span className="history-menu__item-meta-entry">{relativeViewed}</span>
                        )}
                        {viewCountLabel && (
                          <span className="history-menu__item-meta-entry">{viewCountLabel}</span>
                        )}
                      </span>
                    </button>
                  </li>
                );
              })}
            </ul>
          </div>
        )}
        {hasAllSection && (
          <div className="history-menu__section" role="group" aria-label="All apps">
            <div className="history-menu__header">All apps</div>
            <ul className="history-menu__list" role="none">
              {allItems.map((item, indexInSection) => {
                const flatIndex = indexInSection + recentItems.length;
                const { app } = item;
                const label = getAppLabel(app);

                return (
                  <li key={item.key} role="none">
                    <button
                      type="button"
                      role="menuitem"
                      className={clsx('history-menu__item', {
                        'is-active': activeIndex === flatIndex,
                      })}
                      ref={(element) => {
                        itemRefs.current[flatIndex] = element ?? null;
                      }}
                      onClick={() => handleSelect(app)}
                      onKeyDown={handleItemKeyDown(flatIndex)}
                      aria-current={activeIndex === flatIndex ? 'true' : undefined}
                    >
                      <span className="history-menu__item-name">{label}</span>
                    </button>
                  </li>
                );
              })}
            </ul>
          </div>
        )}
        {!hasRecentSection && !hasAllSection && (
          <div className="history-menu__empty" role="status" aria-live="polite">
            No apps match your search.
          </div>
        )}
      </div>
    </div>
  );

  const renderedPanel = usePortal
    ? (portalHost && isOpen ? createPortal(panel, portalHost) : null)
    : (isOpen ? panel : null);

  return (
    <div className={clsx('history-menu', containerClassName)}>
      <button
        type="button"
        ref={buttonRef}
        className={clsx('history-toggle', buttonClassName)}
        onClick={handleToggle}
        onKeyDown={handleToggleKeyDown}
        aria-haspopup="menu"
        aria-expanded={isOpen}
        aria-label="Open app history menu"
        title="Open app history menu"
      >
        <History aria-hidden className={clsx('history-toggle__icon', iconClassName)} />
      </button>
      {renderedPanel}
    </div>
  );
}
