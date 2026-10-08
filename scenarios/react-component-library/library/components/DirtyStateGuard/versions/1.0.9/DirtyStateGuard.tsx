/**
 * @libraryId react-component-library:DirtyStateGuard
 * @displayName DirtyStateGuard
 * @version 1.0.9
 * @tags ["forms","recovery","accessibility","navigation","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:DirtyStateGuard */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import {
  forwardRef,
  useCallback,
  useEffect,
  useId,
  useImperativeHandle,
  useRef,
  useState,
  type CSSProperties,
  type ReactNode,
} from "react";

export type DirtyStateGuardAction = "save" | "discard" | "continue";

export interface DirtyStateGuardHandle {
  requestLeave: () => boolean;
}

export interface DirtyStateGuardPromptProps {
  title: string;
  description: string;
  saveLabel: string;
  discardLabel: string;
  continueLabel: string;
  saving: boolean;
  onSave: () => void;
  onDiscard: () => void;
  onContinue: () => void;
}

export interface DirtyStateGuardProps {
  children: ReactNode;
  isDirty: boolean;
  defaultOpen?: boolean;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  onLeave?: () => void;
  onSave?: () => void | Promise<void>;
  onDiscard?: () => void;
  onAction?: (action: DirtyStateGuardAction) => void;
  title?: string;
  description?: string;
  saveLabel?: string;
  discardLabel?: string;
  continueLabel?: string;
  protectUnload?: boolean;
  renderPrompt?: (props: DirtyStateGuardPromptProps) => ReactNode;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-dirty-guard] { position: relative; }
[data-rcl-dirty-guard-overlay] {
  position: fixed;
  inset: 0;
  box-sizing: border-box;
  z-index: var(--layer-modal);
  display: grid;
  place-items: center;
  padding: var(--space-lg);
  background: var(--color-scrim);
  animation: rcl-dirty-guard-in var(--dur-moderate) var(--ease-standard) both;
}
[data-rcl-dirty-guard-dialog] {
  box-sizing: border-box;
  width: min(100%, 32rem);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-overlay);
  background: var(--color-surface);
  color: var(--color-foreground);
  box-shadow: var(--elev-modal);
  overflow: hidden;
}
[data-rcl-dirty-guard-header] { display: flex; gap: var(--space-sm); padding: var(--space-lg) var(--space-lg) var(--space-sm); }
[data-rcl-dirty-guard-icon] {
  flex: 0 0 auto;
  display: grid;
  place-items: center;
  inline-size: 2.5rem;
  block-size: 2.5rem;
  border-radius: 50%;
  background: var(--color-warning-subtle);
  color: var(--color-warning-foreground);
}
[data-rcl-dirty-guard-title] { margin: 0; font-size: var(--font-size-lg); line-height: 1.25; letter-spacing: -.02em; }
[data-rcl-dirty-guard-copy] { margin: 6px 0 0; color: var(--color-muted-foreground); font-size: var(--font-size-sm); line-height: 1.5; }
[data-rcl-dirty-guard-body] { padding: 0 var(--space-lg) var(--space-md); }
[data-rcl-dirty-guard-note] { margin: 0; padding: var(--space-sm); border-radius: var(--radius-control); background: var(--color-surface-muted); color: var(--color-muted-foreground); font-size: 13px; line-height: 1.45; }
[data-rcl-dirty-guard-actions] { display: flex; flex-wrap: wrap; align-items: center; justify-content: flex-end; gap: var(--space-2xs); padding: var(--space-sm) var(--space-lg) var(--space-lg); }
[data-rcl-dirty-guard-actions] button { min-block-size: 2.75rem; border-radius: var(--radius-control); padding: 0 var(--space-sm); font: inherit; font-size: var(--font-size-sm); font-weight: 700; cursor: pointer; }
[data-rcl-dirty-guard-continue] { border: 1px solid var(--color-border); background: transparent; color: var(--color-foreground); }
[data-rcl-dirty-guard-discard] { border: 1px solid var(--color-danger-border); background: transparent; color: var(--color-danger-foreground); }
[data-rcl-dirty-guard-save] { border: 0; background: var(--color-primary); color: var(--color-primary-foreground); }
[data-rcl-dirty-guard-actions] button:disabled { cursor: wait; opacity: var(--opacity-disabled); }
@keyframes rcl-dirty-guard-in { from { opacity: 0; } to { opacity: 1; } }
@media (max-width: 480px) {
  [data-rcl-dirty-guard-overlay] { align-items: end; padding: 0; }
  [data-rcl-dirty-guard-dialog] { border-radius: var(--radius-overlay) var(--radius-overlay) 0 0; }
  [data-rcl-dirty-guard-actions] { display: grid; grid-template-columns: 1fr; }
  [data-rcl-dirty-guard-actions] button { inline-size: 100%; }
}

`;

const buttonBase: CSSProperties = { minWidth: 96 };

export const DirtyStateGuard = forwardRef<DirtyStateGuardHandle, DirtyStateGuardProps>(
  function DirtyStateGuard(
    {
      children,
      isDirty,
      defaultOpen = false,
      open,
      onOpenChange,
      onLeave,
      onSave,
      onDiscard,
      onAction,
      title,
      description,
      saveLabel = "Save changes",
      discardLabel = "Discard changes",
      continueLabel = "Keep editing",
      protectUnload = true,
      renderPrompt,
      className,
      style,
    },
    ref,
  ) {
    const libraryStrings = useStrings();
    description =
      description ??
      libraryStrings(
        "forms.dirty-state-guard.leave-now-and-your-recent-edits-will-be-lost-sav",
        "Leave now and your recent edits will be lost.",
      );
    title =
      title ??
      libraryStrings(
        "forms.dirty-state-guard.you-have-unsaved-changes",
        "You have unsaved changes",
      );
    const strings = useStrings();
    const titleId = useId();
    const descriptionId = useId();
    const dialogRef = useRef<HTMLDivElement>(null);
    const lastFocused = useRef<HTMLElement | null>(null);
    const [localOpen, setLocalOpen] = useState(defaultOpen);
    const [saving, setSaving] = useState(false);
    const isControlled = open !== undefined;
    const isOpen = isControlled ? open : localOpen;

    const setOpen = useCallback(
      (next: boolean) => {
        if (!isControlled) setLocalOpen(next);
        onOpenChange?.(next);
      },
      [isControlled, onOpenChange],
    );

    const requestLeave = useCallback(() => {
      if (!isDirty) {
        onLeave?.();
        return true;
      }
      if (typeof document !== "undefined") {
        lastFocused.current = document.activeElement as HTMLElement | null;
      }
      setOpen(true);
      return false;
    }, [isDirty, onLeave, setOpen]);

    useImperativeHandle(ref, () => ({ requestLeave }), [requestLeave]);

    useEffect(() => {
      if (!protectUnload || !isDirty || typeof window === "undefined") return;
      const handleBeforeUnload = (event: BeforeUnloadEvent) => {
        event.preventDefault();
      };
      window.addEventListener("beforeunload", handleBeforeUnload);
      return () => window.removeEventListener("beforeunload", handleBeforeUnload);
    }, [isDirty, protectUnload]);

    useEffect(() => {
      if (!isOpen) {
        lastFocused.current?.focus();
        return;
      }
      dialogRef.current?.focus();
      const handleKeyDown = (event: KeyboardEvent) => {
        if (event.key === "Escape" && !saving) {
          event.preventDefault();
          setOpen(false);
          onAction?.("continue");
        }
      };
      document.addEventListener("keydown", handleKeyDown);
      return () => document.removeEventListener("keydown", handleKeyDown);
    }, [isOpen, onAction, saving, setOpen]);

    const finish = (action: DirtyStateGuardAction) => {
      onAction?.(action);
      if (action === "continue") {
        setOpen(false);
        return;
      }
      if (action === "discard") {
        onDiscard?.();
        setOpen(false);
        onLeave?.();
      }
    };

    const save = async () => {
      setSaving(true);
      try {
        await onSave?.();
        onAction?.("save");
        setOpen(false);
        onLeave?.();
      } finally {
        setSaving(false);
      }
    };

    const promptProps: DirtyStateGuardPromptProps = {
      title,
      description,
      saveLabel,
      discardLabel,
      continueLabel,
      saving,
      onSave: () => void save(),
      onDiscard: () => finish("discard"),
      onContinue: () => finish("continue"),
    };

    return (
      <div
        data-rcl-dirty-guard
        className={className}
        style={style}
        data-dirty={isDirty ? "true" : "false"}
      >
        <StyleSheet libraryId="react-component-library:DirtyStateGuard" version="1.0.9" css={styles} />
        {children}
        {isOpen &&
          (renderPrompt ? (
            renderPrompt(promptProps)
          ) : (
            <div
              data-rcl-dirty-guard-overlay
              role="presentation"
              onMouseDown={(event) => {
                if (event.target === event.currentTarget && !saving) setOpen(false);
              }}
            >
              <div
                ref={dialogRef}
                data-rcl-dirty-guard-dialog
                role="alertdialog"
                aria-modal="true"
                aria-labelledby={titleId}
                aria-describedby={descriptionId}
                tabIndex={-1}
              >
                <div data-rcl-dirty-guard-header>
                  <div data-rcl-dirty-guard-icon aria-hidden="true">
                    !
                  </div>
                  <div>
                    <h2 id={titleId} data-rcl-dirty-guard-title>
                      {title}
                    </h2>
                    <p id={descriptionId} data-rcl-dirty-guard-copy>
                      {description}
                    </p>
                  </div>
                </div>
                <div data-rcl-dirty-guard-body>
                  <p data-rcl-dirty-guard-note>
                    {strings(
                      "forms.dirty-state-guard.saving-preserves-your-work-discarding-cannot-be-",
                      "Saving preserves your work. Discarding cannot be undone.",
                    )}
                  </p>
                </div>
                <div data-rcl-dirty-guard-actions>
                  <button
                    data-testid="forms.dirty-state-guard"
                    type="button"
                    data-rcl-dirty-guard-continue
                    style={buttonBase}
                    disabled={saving}
                    onClick={promptProps.onContinue}
                  >
                    {continueLabel}
                  </button>
                  <button
                    data-testid="forms.dirty-state-guard"
                    type="button"
                    data-rcl-dirty-guard-discard
                    style={buttonBase}
                    disabled={saving}
                    onClick={promptProps.onDiscard}
                  >
                    {discardLabel}
                  </button>
                  <button
                    data-testid="forms.dirty-state-guard"
                    type="button"
                    data-rcl-dirty-guard-save
                    style={buttonBase}
                    disabled={saving}
                    onClick={promptProps.onSave}
                  >
                    {saving ? "Saving…" : saveLabel}
                  </button>
                </div>
              </div>
            </div>
          ))}
      </div>
    );
  },
);
