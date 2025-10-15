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
}

export default function HistoryMenu({ recentApps, allApps, shouldShowHistory, onSelect }: HistoryMenuProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const [menuCoords, setMenuCoords] = useState<{ top: number; left: number } | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const buttonRef = useRef<HTMLButtonElement | null>(null);
  const menuRef = useRef<HTMLDivElement | null>(null);
  const searchInputRef = useRef<HTMLInputElement | null>(null);
  const itemRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const fallbackKeyMap = useRef(new WeakMap<App, string>());
  const fallbackKeyCounter = useRef(0);
  const isBrowser = typeof document !== 'undefined';

  const normalizedQuery = searchQuery.trim().toLowerCase();

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

  const updateMenuPosition = useCallback(() => {
    if (!isBrowser) {
      return;
    }

    const button = buttonRef.current;
    const menu = menuRef.current;
    if (!button || !menu) {
      return;
    }

    const buttonRect = button.getBoundingClientRect();
    const menuRect = menu.getBoundingClientRect();
    const menuWidth = menuRect.width || menu.offsetWidth;
    const viewportWidth = window.innerWidth;

    const idealLeft = buttonRect.right - menuWidth;
    const minLeft = HISTORY_MENU_MARGIN;
    const maxLeft = Math.max(minLeft, viewportWidth - menuWidth - HISTORY_MENU_MARGIN);
    const clampedLeft = Math.min(Math.max(idealLeft, minLeft), maxLeft);

    const top = Math.round(buttonRect.bottom + HISTORY_MENU_OFFSET);
    const left = Math.round(clampedLeft);

    setMenuCoords({ top, left });
  }, [isBrowser]);

  const menuStyle = useMemo<CSSProperties>(() => {
    if (!menuCoords) {
      return {
        top: '-9999px',
        left: '-9999px',
        visibility: 'hidden',
      };
    }

    return {
      top: `${menuCoords.top}px`,
      left: `${menuCoords.left}px`,
      visibility: 'visible',
    };
  }, [menuCoords]);

  useEffect(() => {
    if (!shouldShowHistory && isOpen) {
      setIsOpen(false);
      setSearchQuery('');
    }
  }, [isOpen, shouldShowHistory]);

  useEffect(() => {
    if (!isOpen) {
      setMenuCoords(null);
      itemRefs.current = [];
      return;
    }

    const frame = window.requestAnimationFrame(() => {
      updateMenuPosition();
    });

    const handleResizeOrScroll = () => {
      updateMenuPosition();
    };

    window.addEventListener('resize', handleResizeOrScroll);
    window.addEventListener('scroll', handleResizeOrScroll, true);

    return () => {
      window.cancelAnimationFrame(frame);
      window.removeEventListener('resize', handleResizeOrScroll);
      window.removeEventListener('scroll', handleResizeOrScroll, true);
    };
  }, [isOpen, totalItems, hasRecentSection, hasAllSection, updateMenuPosition]);

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

      setIsOpen(false);
      setMenuCoords(null);
      setActiveIndex(null);
      setSearchQuery('');
      buttonRef.current?.focus();
    };

    const handleKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        setIsOpen(false);
        setMenuCoords(null);
        setActiveIndex(null);
        setSearchQuery('');
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
  }, [isOpen]);

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

    setIsOpen(prev => {
      const next = !prev;
      if (!next) {
        setSearchQuery('');
        setActiveIndex(null);
      }
      return next;
    });
  }, [shouldShowHistory]);

  const handleToggleKeyDown = useCallback((event: ReactKeyboardEvent<HTMLButtonElement>) => {
    if (!shouldShowHistory) {
      return;
    }

    if (event.key === 'ArrowDown') {
      event.preventDefault();
      setActiveIndex(totalItems > 0 ? 0 : null);
      setIsOpen(true);
      return;
    }

    if (event.key === 'ArrowUp') {
      event.preventDefault();
      setActiveIndex(totalItems > 0 ? totalItems - 1 : null);
      setIsOpen(true);
      return;
    }

    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      setIsOpen(prev => {
        const next = !prev;
        if (!next) {
          setSearchQuery('');
          setActiveIndex(null);
        }
        return next;
      });
    }
  }, [shouldShowHistory, totalItems]);

  const handleSelect = useCallback((app: App) => {
    setIsOpen(false);
    setMenuCoords(null);
    setActiveIndex(null);
    setSearchQuery('');
    onSelect(app);
  }, [onSelect]);

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
      setIsOpen(false);
      setMenuCoords(null);
      setActiveIndex(null);
      setSearchQuery('');
      buttonRef.current?.focus();
    }
  }, [focusItem, totalItems]);

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
      setIsOpen(false);
      setMenuCoords(null);
      setActiveIndex(null);
      setSearchQuery('');
      buttonRef.current?.focus();
    }
  }, [focusItem, totalItems]);

  if (!shouldShowHistory) {
    return null;
  }

  return (
    <div className="history-menu">
      <button
        type="button"
        ref={buttonRef}
        className="history-toggle"
        onClick={handleToggle}
        onKeyDown={handleToggleKeyDown}
        aria-haspopup="menu"
        aria-expanded={isOpen}
        aria-label="Open app history menu"
        title="Open app history menu"
      >
        <History aria-hidden className="history-toggle__icon" />
      </button>
      {isBrowser && isOpen && createPortal(
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
        </div>,
        document.body,
      )}
    </div>
  );
}
