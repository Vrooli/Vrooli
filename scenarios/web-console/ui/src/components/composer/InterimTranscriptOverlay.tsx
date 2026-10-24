import { useEffect, useLayoutEffect, useRef, useState, type CSSProperties, type RefObject } from "react";
import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
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
  /** Native voice identifies the speaker; ordinary dictation defaults to user. */
  /** Live agent captions are a conversation, never a pending draft. */
  readonly liveAgent?: boolean;
  readonly connecting?: boolean;
  readonly speakerRole?: "user" | "assistant" | null;
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
  speakerRole,
  liveAgent = false,
  connecting = false,
  textareaRef,
  className,
  style,
  testId,
}: InterimTranscriptOverlayProps) {
  const { t } = useTranslation();
  const active = liveAgent || interim.length > 0;
  const liveScrollRef = useRef<HTMLDivElement>(null);
  const overlayRef = useRef<HTMLDivElement>(null);
  const userScrolledAwayRef = useRef(false);
  const [value, setValue] = useState(() => (active ? draft.getValue() : ""));

  useEffect(() => {
    if (!active) {
      userScrolledAwayRef.current = false;
      return;
    }
    setValue(draft.getValue());
    return draft.subscribe((change) => { setValue(change.value); });
  }, [active, draft]);

  // Follow the textarea's scroll so a long draft stays in register. While the
  // operator remains at the bottom, keep the growing hypothesis visible. A
  // deliberate scroll-away is respected until the operator returns to the
  // bottom, so dictation never yanks a reader back to the live tail.
  useLayoutEffect(() => {
    const textarea = textareaRef.current;
    const overlay = overlayRef.current;
    if (!active || liveAgent || !textarea || !overlay) return;
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
    return () => { textarea.removeEventListener("scroll", sync); };
  }, [active, liveAgent, textareaRef, value, interim]);

  // Native captions have no textarea content to measure. Follow their own
  // scroll height, while respecting a reader who scrolls back through speech.
  useLayoutEffect(() => {
    const scroller = liveScrollRef.current;
    if (!liveAgent || !scroller) return;
    if (!userScrolledAwayRef.current) scroller.scrollTop = scroller.scrollHeight;
  }, [liveAgent, interim, speakerRole]);


  if (!active) return null;

  if (liveAgent) return (
    <div ref={overlayRef} role="region" aria-label={t(connecting ? strings.agentVoice.connecting : strings.agentVoice.liveAutomatic)}
      data-testid={testId ?? "composer-interim-overlay"}
      className={`absolute inset-0 z-20 flex min-h-0 flex-col ${className}`} style={style}>
      <span className="shrink-0 text-[10px] leading-4 text-wc-text-muted">{t(connecting ? strings.agentVoice.connecting : strings.agentVoice.liveAutomatic)}</span>
      <div ref={liveScrollRef} data-testid="agent-voice-caption-scroll" className="min-h-0 flex-1 overflow-y-auto whitespace-pre-wrap break-words"
        onScroll={(event) => {
          const el = event.currentTarget;
          userScrolledAwayRef.current = el.scrollHeight - el.scrollTop - el.clientHeight > 8;
        }}>
        <span data-testid={testId ? `${testId}-text` : "composer-interim"}
          data-speaker={speakerRole ?? "user"}
          className={speakerRole === "assistant" ? "text-wc-text-secondary" : "text-wc-accent"}>
          {interim ? <><span className="font-medium">{t(speakerRole === "assistant" ? strings.agentVoice.agentCaption : strings.agentVoice.userCaption)}: </span>{interim}</> : t(connecting ? strings.agentVoice.connecting : strings.agentVoice.listening)}
        </span>
      </div>
    </div>
  );

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
        data-speaker={speakerRole ?? "user"}
        className={`border-b border-dotted ${speakerRole === "assistant" ? "border-wc-text-secondary text-wc-text-secondary" : "border-wc-accent text-wc-accent"}`}
      >
        {separator}
        {interim}
      </span>
    </div>
  );
}
