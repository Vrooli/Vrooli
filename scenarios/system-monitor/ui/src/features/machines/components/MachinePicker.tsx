// DOC: docs/reference/cross-platform-effort/machine-linking-ux-2026-08-26.html#screen-05
//
// One menu, and the app has a fleet. The control sets the subject of every
// panel below it, so it carries three things a native <select> cannot:
// reachability per row, the grant the caller holds on each machine, and the
// entry point for linking a new one. "Add a machine…" lives here deliberately —
// if linking is only reachable from vrooli-bridge, every app that wants a
// second machine inherits a detour through an app the user has no other reason
// to open.
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ChevronDown, Check, Plus } from 'lucide-react';
import { useClickOutside } from '../../../shared/hooks/useClickOutside';
import { useEscapeKey } from '../../../hooks/useEscapeKey';
import { describeMachine, sortMachinesForPicker } from '../presence';
import type { Machine } from '../../../types';

interface MachinePickerProps {
  machines: Machine[];
  selectedMachineID: string;
  onSelectMachine: (machineID: string) => void;
  onAddMachine?: () => void;
}

export const MachinePicker = ({
  machines,
  selectedMachineID,
  onSelectMachine,
  onAddMachine
}: MachinePickerProps) => {
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const rootRef = useRef<HTMLDivElement | null>(null);
  const listRef = useRef<HTMLDivElement | null>(null);
  const triggerRef = useRef<HTMLButtonElement | null>(null);

  // Closing returns focus to the control that opened the menu; otherwise a
  // keyboard reader who presses Escape is left with focus on nothing.
  const close = useCallback(() => {
    setOpen(false);
    triggerRef.current?.focus();
  }, []);

  useClickOutside(rootRef, close, open);
  useEscapeKey(close, open);

  const ordered = useMemo(() => sortMachinesForPicker(machines), [machines]);
  const selected = ordered.find(machine => machine.id === selectedMachineID) ?? ordered[0];
  const selectedPresence = selected ? describeMachine(selected) : null;

  useEffect(() => {
    if (!open) return;
    const active = listRef.current?.querySelector<HTMLElement>('[data-active="true"]');
    active?.focus();
  }, [open, activeIndex]);

  if (ordered.length === 0) {
    return null;
  }

  const choose = (machineID: string) => {
    onSelectMachine(machineID);
    close();
  };

  const handleListKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'ArrowDown') {
      event.preventDefault();
      setActiveIndex(prev => (prev + 1) % ordered.length);
    } else if (event.key === 'ArrowUp') {
      event.preventDefault();
      setActiveIndex(prev => (prev - 1 + ordered.length) % ordered.length);
    } else if (event.key === 'Home') {
      event.preventDefault();
      setActiveIndex(0);
    } else if (event.key === 'End') {
      event.preventDefault();
      setActiveIndex(ordered.length - 1);
    }
  };

  return (
    <div ref={rootRef} className="machine-picker">
      <button
        ref={triggerRef}
        type="button"
        className="machine-picker__trigger"
        data-testid="machine-picker"
        data-remote={selectedPresence && !selectedPresence.isLocal ? 'true' : 'false'}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={`Machine: ${selected?.name ?? 'none'}. Change the machine this app is reading.`}
        onClick={() => {
          // Open onto the current selection, set before the menu mounts, so the
          // first arrow keypress moves from where the reader already is.
          const index = ordered.findIndex(machine => machine.id === selectedMachineID);
          setActiveIndex(index >= 0 ? index : 0);
          setOpen(prev => !prev);
        }}
      >
        <span className={`machine-led machine-led--${selectedPresence?.tone ?? 'local'}`} aria-hidden="true" />
        <span className="machine-picker__name">{selected?.name}</span>
        <ChevronDown size={13} aria-hidden="true" className="machine-picker__caret" />
      </button>

      {open && (
        <div className="machine-picker__menu">
          <div className="machine-picker__eyebrow">Viewing</div>

          {/* The listbox is exactly the options and nothing else: a wrapper
              between it and its options, or the linking button inside it,
              would break the relationship assistive tech relies on.
              The list scrolls; the linking action does not — a fleet large
              enough to scroll is exactly the case where "Add a machine…" must
              not scroll out of reach. */}
          <div
            ref={listRef}
            className="machine-picker__options"
            role="listbox"
            aria-label="Machine"
            onKeyDown={handleListKeyDown}
          >
          {ordered.map((machine, index) => {
            const presence = describeMachine(machine);
            const isSelected = machine.id === selectedMachineID;
            return (
              <button
                key={machine.id || 'local'}
                type="button"
                role="option"
                aria-selected={isSelected}
                tabIndex={index === activeIndex ? 0 : -1}
                data-active={index === activeIndex}
                data-tone={presence.tone}
                className="machine-picker__option"
                onClick={() => { choose(machine.id); }}
                onFocus={() => { setActiveIndex(index); }}
              >
                <span className={`machine-led machine-led--${presence.tone}`} aria-hidden="true" />
                <span className="machine-picker__option-body">
                  <span className="machine-picker__option-name">{machine.name}</span>
                  <span className="machine-picker__option-meta">{presence.meta}</span>
                </span>
                {isSelected ? (
                  <Check size={14} aria-hidden="true" className="machine-picker__check" />
                ) : presence.grantLabel ? (
                  <span className="machine-chip">{presence.grantLabel}</span>
                ) : null}
              </button>
            );
          })}
          </div>

          {onAddMachine && (
            <button
              type="button"
              className="machine-picker__add"
              data-testid="add-machine"
              onClick={() => { close(); onAddMachine(); }}
            >
              <Plus size={14} aria-hidden="true" />
              Add a machine&hellip;
            </button>
          )}
        </div>
      )}
    </div>
  );
};
