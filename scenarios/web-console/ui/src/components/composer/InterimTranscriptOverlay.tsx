import { useEffect, useLayoutEffect, useRef, useState, type CSSProperties, type RefObject } from "react";
import { transcriptSeparator } from "@vrooli/audio-capture-browser";
import type { ComposerDraft } from "../../hooks/useComposerDraft";

export interface InterimTranscriptOverlayProps {
  /** The draft whose settled text sits before the interim span. */
  readonly draft: ComposerDraft;
  /**
   * The unsettled hypothesis to preview. Already has the committed prefix
   * subtracted by `TranscriptBuffer` in the capture package, so it is appended
   * verbatim rather than reconciled here.
   */
  readonly interim: string;
  /** The textarea this overlay mirrors; used to follow its scroll position. */
  readonly textareaRef: RefObject<HTMLTextAreaElement>;
  /**
   * Typography and box metrics identical to the textarea's. Any divergence
   * shows up as text that drifts out of register as the draft grows.
   */
  readonly className: string;
  /**
   * Box metrics that cannot be expressed as a class because they read the
   * host field's own custom properties. The padding here must resolve to the
   * same values the textarea receives, so both are sourced from one token
   * rather than hand-matched.
   */
  readonly style?: CSSProperties;
  readonly testId?: string;
}

/**
 * Draws settled draft text plus the live dictation hypothesis underneath a
 * transparent textarea, so unsettled words appear in the input as they are
 * spoken without a `contenteditable` (which would cost IME composition, mobile
 * keyboards, and native selection).
 *
 * Renders nothing when there is no interim text. That is deliberate: the
 * collapsed toolbar input is uncontrolled precisely so typing never re-renders
 * React, and subscribing to the draft only while a hypothesis is on screen
 * keeps that property for every non-dictating keystroke.
 *
 * The mirror is `aria-hidden`. Interim text is revised several times per second
 * and a live region over it would talk continuously; the settled text reaches
 * assistive technology through the textarea itself once a segment commits.
 */
export default function InterimTranscriptOverlay({
  draft,
  interim,
  textareaRef,
  className,
  style,
  testId,
}: InterimTranscriptOverlayProps) {
  const active = interim.length > 0;
  const overlayRef = useRef<HTMLDivElement>(null);
  const userScrolledAwayRef = useRef(false);
  const [value, setValue] = useState(() => (active ? draft.getValue() : ""));

  useEffect(() => {
    if (!active) {
      userScrolledAwayRef.current = false;
      return;
    }
    setValue(draft.getValue());
    return draft.subscribe((change) => setValue(change.value));
  }, [active, draft]);

  // Follow the textarea's scroll so a long draft stays in register. While the
  // operator remains at the bottom, keep the growing hypothesis visible. A
  // deliberate scroll-away is respected until the operator returns to the
  // bottom, so dictation never yanks a reader back to the live tail.
  useLayoutEffect(() => {
    const textarea = textareaRef.current;
    const overlay = overlayRef.current;
    if (!active || !textarea || !overlay) return;
    const sync = () => {
      const remaining = textarea.scrollHeight - textarea.scrollTop - textarea.clientHeight;
      userScrolledAwayRef.current = remaining > 8;
      overlay.scrollTop = textarea.scrollTop;
    };
    if (!userScrolledAwayRef.current) {
      textarea.scrollTop = textarea.scrollHeight;
    }
    sync();
    textarea.addEventListener("scroll", sync, { passive: true });
    return () => textarea.removeEventListener("scroll", sync);
  }, [active, textareaRef, value, interim]);

  if (!active) return null;

  // Same rule the transcript buffer will apply when this text settles, so the
  // preview never shifts by a space at the moment it commits.
  const separator = transcriptSeparator(value, interim);

  return (
    <div
      ref={overlayRef}
      aria-hidden="true"
      data-testid={testId ?? "composer-interim-overlay"}
      className={`pointer-events-none absolute inset-0 z-0 overflow-hidden whitespace-pre-wrap break-words ${className}`}
      style={style}
    >
      <span>{value}</span>
      <span
        data-testid={testId ? `${testId}-text` : "composer-interim"}
        className="border-b border-dotted border-wc-accent text-wc-accent"
      >
        {separator}
        {interim}
      </span>
    </div>
  );
}
