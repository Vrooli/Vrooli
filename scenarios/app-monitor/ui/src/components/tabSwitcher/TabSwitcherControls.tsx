import type { RefObject } from 'react';
import { Loader2, Search, Shuffle, X } from 'lucide-react';
import clsx from 'clsx';
import type { ShortcutState } from '@/utils/tabSwitcherShortcut';
import { SEGMENTS, type SegmentId } from './tabSwitcherConstants';

export function TabSwitcherHeader({
  title,
  shortcut,
  onClose,
}: {
  title: string;
  shortcut: ShortcutState | null;
  onClose: () => void;
}) {
  return (
    <header className="tab-switcher__header">
      <div className="tab-switcher__header-text">
        <h2>{title}</h2>
        {shortcut ? (
          <p>
            Tip: press <ShortcutChip shortcut={shortcut} /> to toggle this dialog.
          </p>
        ) : null}
      </div>
      <button
        type="button"
        className="tab-switcher__close"
        aria-label="Close tab switcher"
        onClick={onClose}
      >
        <X aria-hidden />
      </button>
    </header>
  );
}

function ShortcutChip({ shortcut }: { shortcut: ShortcutState }) {
  return (
    <span className="tab-switcher__shortcut-chip">
      <span className="visually-hidden">{shortcut.description}</span>
      <span className="tab-switcher__shortcut-visual" aria-hidden="true">
        {shortcut.keys.map((key, index) => (
          <span key={`${key}-${index}`} className="tab-switcher__shortcut-group">
            <span className="tab-switcher__shortcut-key">{key}</span>
            {index < shortcut.keys.length - 1 ? (
              <span className="tab-switcher__shortcut-plus" aria-hidden="true">+</span>
            ) : null}
          </span>
        ))}
      </span>
    </span>
  );
}

export function TabSwitcherControls({
  activeSegment,
  onSegmentSelect,
  search,
  onSearchChange,
  onSearchClear,
  onSearchEnter,
  searchInputRef,
  showAutoNext,
  isAutoNextRunning,
  onAutoNext,
  disableAutoNext,
}: {
  activeSegment: SegmentId;
  onSegmentSelect: (segmentId: SegmentId) => void;
  search: string;
  onSearchChange: (value: string) => void;
  onSearchClear: () => void;
  onSearchEnter: () => void;
  searchInputRef: RefObject<HTMLInputElement>;
  showAutoNext: boolean;
  isAutoNextRunning: boolean;
  onAutoNext: () => void;
  disableAutoNext: boolean;
}) {
  return (
    <div className="tab-switcher__controls">
      <TabSwitcherSearch
        value={search}
        onChange={onSearchChange}
        onClear={onSearchClear}
        onEnter={onSearchEnter}
        inputRef={searchInputRef}
      />
      <SegmentSelector
        activeSegment={activeSegment}
        onSelect={onSegmentSelect}
      />
      {showAutoNext && (
        <AutoNextButton
          isRunning={isAutoNextRunning}
          onClick={onAutoNext}
          disabled={disableAutoNext}
        />
      )}
    </div>
  );
}

function TabSwitcherSearch({
  value,
  onChange,
  onClear,
  onEnter,
  inputRef,
}: {
  value: string;
  onChange: (value: string) => void;
  onClear: () => void;
  onEnter: () => void;
  inputRef: RefObject<HTMLInputElement>;
}) {
  return (
    <div className="tab-switcher__search">
      <Search size={16} aria-hidden />
      <input
        type="text"
        ref={inputRef}
        value={value}
        onChange={event => onChange(event.target.value)}
        onKeyDown={(event) => {
          if (event.key !== 'Enter' || event.shiftKey || event.altKey || event.ctrlKey || event.metaKey || event.nativeEvent.isComposing) {
            return;
          }
          event.preventDefault();
          onEnter();
        }}
        placeholder="Search scenarios or resources"
        aria-label="Search"
      />
      {value && (
        <button type="button" onClick={onClear} aria-label="Clear search">
          <X size={14} aria-hidden />
        </button>
      )}
    </div>
  );
}

function SegmentSelector({
  activeSegment,
  onSelect,
}: {
  activeSegment: SegmentId;
  onSelect: (segmentId: SegmentId) => void;
}) {
  return (
    <div className="tab-switcher__segment" role="tablist" aria-label="Tab segments">
      {SEGMENTS.map((segment) => {
        const Icon = segment.icon;
        const isActive = segment.id === activeSegment;
        return (
          <button
            key={segment.id}
            type="button"
            role="tab"
            aria-selected={isActive}
            tabIndex={isActive ? 0 : -1}
            className={clsx('tab-switcher__segment-btn', isActive && 'active')}
            onClick={() => onSelect(segment.id)}
            aria-label={segment.label}
          >
            <Icon size={16} aria-hidden />
            <span className="tab-switcher__segment-label">{segment.label}</span>
          </button>
        );
      })}
    </div>
  );
}

function AutoNextButton({
  isRunning,
  onClick,
  disabled,
}: {
  isRunning: boolean;
  onClick: () => void;
  disabled: boolean;
}) {
  return (
    <button
      type="button"
      className="tab-switcher__auto-next"
      onClick={onClick}
      disabled={isRunning || disabled}
    >
      <span className="tab-switcher__auto-next-icon">
        {isRunning ? (
          <Loader2 size={24} aria-hidden className="tab-switcher__auto-next-spinner" />
        ) : (
          <Shuffle size={24} aria-hidden />
        )}
      </span>
      <span className="tab-switcher__auto-next-text">
        {isRunning ? 'Selecting next scenario…' : 'Auto-next scenario'}
      </span>
    </button>
  );
}
