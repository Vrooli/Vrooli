import type { RefObject } from 'react';
import { Loader2, Search, Shuffle, X } from 'lucide-react';
import clsx from 'clsx';
import { SEGMENTS, type SegmentId } from './tabSwitcherConstants';

export function TabSwitcherHeader({ title, onClose }: { title: string; onClose: () => void }) {
  return (
    <header className="tab-switcher__header">
      <div className="tab-switcher__header-text">
        <h2>{title}</h2>
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

export function TabSwitcherControls({
  activeSegment,
  onSegmentSelect,
  search,
  onSearchChange,
  onSearchClear,
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
  inputRef,
}: {
  value: string;
  onChange: (value: string) => void;
  onClear: () => void;
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
    <div className="tab-switcher__segment">
      {SEGMENTS.map(segment => {
        const Icon = segment.icon;
        const isActive = segment.id === activeSegment;
        return (
          <button
            key={segment.id}
            type="button"
            className={clsx('tab-switcher__segment-btn', isActive && 'active')}
            onClick={() => onSelect(segment.id)}
            aria-pressed={isActive}
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
