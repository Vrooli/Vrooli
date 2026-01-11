import { FormEvent, type RefObject } from 'react';
import { Globe, Loader2, Plus, Search, Shuffle, X } from 'lucide-react';
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
  showWebForm,
  webFormValue,
  webFormError,
  webFormErrorId,
  onWebFormChange,
  onWebFormSubmit,
  disableWebFormSubmit,
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
  showWebForm: boolean;
  webFormValue: string;
  webFormError: string | null;
  webFormErrorId: string;
  onWebFormChange: (value: string) => void;
  onWebFormSubmit: (event: FormEvent<HTMLFormElement>) => void;
  disableWebFormSubmit: boolean;
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
      {showWebForm && (
        <WebTabForm
          value={webFormValue}
          onChange={onWebFormChange}
          onSubmit={onWebFormSubmit}
          error={webFormError}
          errorId={webFormErrorId}
          disableSubmit={disableWebFormSubmit}
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
        placeholder="Search scenarios, resources, or tabs"
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

function WebTabForm({
  value,
  onChange,
  onSubmit,
  error,
  errorId,
  disableSubmit,
}: {
  value: string;
  onChange: (value: string) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  error: string | null;
  errorId: string;
  disableSubmit: boolean;
}) {
  return (
    <>
      <form className="tab-switcher__web-form" onSubmit={onSubmit}>
        <Globe size={16} aria-hidden />
        <input
          type="text"
          inputMode="url"
          autoComplete="off"
          autoCorrect="off"
          autoCapitalize="none"
          spellCheck={false}
          value={value}
          onChange={event => onChange(event.target.value)}
          placeholder="https://example.com"
          aria-label="Open a new web tab"
          aria-invalid={error ? 'true' : 'false'}
          aria-describedby={error ? errorId : undefined}
        />
        {value && (
          <button
            type="button"
            className="tab-switcher__clear"
            onClick={() => onChange('')}
            aria-label="Clear URL"
          >
            <X size={14} aria-hidden />
          </button>
        )}
        <button
          type="submit"
          className="tab-switcher__web-submit"
          disabled={disableSubmit}
        >
          <Plus size={16} aria-hidden />
          <span>Add tab</span>
        </button>
      </form>
      {error && (
        <p className="tab-switcher__web-error" role="alert" id={errorId}>
          {error}
        </p>
      )}
    </>
  );
}
