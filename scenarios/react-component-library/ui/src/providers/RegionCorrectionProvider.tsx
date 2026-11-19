import { useCallback, useEffect, useRef, useState, type MouseEvent, type PointerEvent, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { getLayerPortalContainer, layerManager } from "@vrooli/react-component-library/LayerManager/2.0.2";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { useTranslation } from "../i18n";

export const REGION_CORRECTION_SCHEMA = "ux-region-feedback/v1" as const;
export const REGION_LONG_PRESS_MS = 650;

const correctionStyles = `
[data-rcl-region-correction-root] { display: contents; }
[data-rcl-region-correction-launcher] { position: fixed; z-index: 90; inset-inline-end: var(--space-sm); inset-block-end: var(--space-sm); display: flex; max-inline-size: calc(100vw - 1.5rem); flex-direction: column; align-items: flex-end; gap: var(--space-2xs); }
[data-rcl-region-correction-instruction] { margin: 0; border-radius: var(--radius-control); background: var(--color-foreground); padding: var(--space-2xs) var(--space-sm); color: var(--color-surface); font: var(--text-body-sm); box-shadow: var(--elev-raised); }
[data-rcl-region-correction-feedback] { max-inline-size: 24rem; margin: 0; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface-raised); padding: var(--space-2xs) var(--space-sm); color: var(--color-foreground); font: var(--text-body-sm); box-shadow: var(--elev-raised); }
[data-rcl-region-correction-launcher] button { min-block-size: var(--tap-target-min); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-pill); background: var(--color-surface-raised); padding-inline: var(--space-sm); color: var(--color-foreground); font: var(--text-label); box-shadow: var(--elev-raised); }
[data-rcl-region-correction-backdrop] { position: fixed; z-index: 100; inset: 0; display: flex; align-items: flex-end; justify-content: center; background: color-mix(in srgb, var(--color-foreground) 42%, transparent); }
[data-rcl-region-correction-panel] { box-sizing: border-box; inline-size: 100%; max-block-size: min(90dvh, 48rem); overflow-y: auto; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel) var(--radius-panel) 0 0; background: var(--color-surface-raised); padding: var(--space-md); color: var(--color-foreground); box-shadow: var(--elev-overlay); }
[data-rcl-region-correction-header] { display: flex; align-items: flex-start; justify-content: space-between; gap: var(--space-sm); }
[data-rcl-region-correction-header] h2 { margin: 0; font: var(--text-title); }
[data-rcl-region-correction-header] p { margin-block: var(--space-2xs) 0; color: var(--color-muted-foreground); font: var(--text-body-sm); }
[data-rcl-region-correction-close] { min-inline-size: var(--tap-target-min); min-block-size: var(--tap-target-min); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-foreground); }
[data-rcl-region-correction-crop] { margin-block: var(--space-sm); overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); }
[data-rcl-region-correction-crop] img { display: block; inline-size: 100%; max-block-size: 10rem; object-fit: contain; }
[data-rcl-region-correction-crop] figcaption { padding: var(--space-2xs) var(--space-xs); color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-region-correction-context] { display: grid; grid-template-columns: 1fr 2fr; gap: var(--space-2xs); margin-block: var(--space-sm); border-radius: var(--radius-control); background: var(--color-surface-muted); padding: var(--space-xs); font: var(--text-caption); }
[data-rcl-region-correction-context] dd { margin: 0; overflow-wrap: anywhere; }
[data-rcl-region-correction-panel] label { font: var(--text-label); }
[data-rcl-region-correction-panel] textarea { box-sizing: border-box; display: block; inline-size: 100%; min-block-size: 6rem; margin-block-start: var(--space-2xs); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-field); padding: var(--space-xs); color: var(--color-foreground); font: var(--text-body); }
[data-rcl-region-correction-status] { min-block-size: 1.5rem; margin-block: var(--space-2xs); color: var(--color-muted-foreground); font: var(--text-body-sm); }
[data-rcl-region-correction-submit] { inline-size: 100%; min-block-size: var(--tap-target-min); border: 0; border-radius: var(--radius-control); background: var(--color-primary); padding-inline: var(--space-sm); color: var(--color-on-primary); font: var(--text-label); }
[data-rcl-region-correction-submit]:disabled { opacity: .55; }
[data-rcl-region-correction-shortcut] { margin-block: var(--space-2xs) 0; color: var(--color-muted-foreground); font: var(--text-caption); }
@media (min-width: 40rem) { [data-rcl-region-correction-backdrop] { align-items: center; } [data-rcl-region-correction-panel] { max-inline-size: 32rem; border-radius: var(--radius-panel); } }
`;

export interface RegionCorrectionEvidence {
  schema: typeof REGION_CORRECTION_SCHEMA;
  kind: "correction";
  correction_id: string;
  origin: { kind: "served-page"; id: string } | { kind: "hub-packet"; id: string; attachment: string };
  owner_scenario: string;
  corpus: string;
  storage: "local";
  private: true;
  capture: { id: string; build_id: string; route: string; state_id: string; data_id: string; fidelity: "captured" | "composited" | "rendered" | "sketch"; viewport: { width: number; height: number }; image: { path: string; sha256: string } };
  region: { id: string; node_id: string; label: string; component: string; evidence_id: string; crop: { path: string; sha256: string }; bounds: { x: number; y: number; width: number; height: number } };
}

export interface ResolvedRegionCorrection {
  evidence: RegionCorrectionEvidence;
  /** Same-origin, locally served preview; never included in the submitted correction. */
  cropPreviewUrl: string;
}

export interface RegionCorrectionProviderProps {
  children: ReactNode;
  /** Resolve identity and a locally stored crop from host instrumentation. Must not return UI text or form values. */
  resolveEvidence: (element: HTMLElement) => Promise<ResolvedRegionCorrection | null>;
  /** Route to the scenario owner and private local EX1 intake. */
  submitCorrection: (input: RegionCorrectionEvidence & { text: string }) => Promise<void>;
  shortcut?: string;
  disabled?: boolean;
}

interface Point { x: number; y: number; element: HTMLElement }

function targetElement(target: EventTarget | null): HTMLElement | null {
  if (!(target instanceof Element)) return null;
  if (target.closest("[data-ux-provider-ui]")) return null;
  const explicit = target.closest<HTMLElement>("[data-ux-feedback-target]");
  if (explicit) return explicit;
  const node = target.closest<HTMLElement>("[data-ux-node-id][data-ux-component]");
  return node;
}

function isInteractive(element: HTMLElement): boolean {
  return Boolean(element.closest("button,a,input,textarea,select,[contenteditable=true],[role=button]"));
}

function explicitlyMarked(element: HTMLElement): boolean {
  return element.matches("[data-ux-feedback-target]");
}

const stableId = (value: string): boolean => /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(value);
const digest = (value: string): boolean => /^[0-9a-f]{64}$/.test(value);
const bundlePath = (value: string): boolean => Boolean(value && /^[A-Za-z0-9._~!$&'()*+,;=@%/-]+$/.test(value) && value.split("/").every((part) => part !== "" && part !== "." && part !== ".."));

/**
 * Evidence arrives from host instrumentation, which sits outside this module's
 * type boundary: a host can be older than the panel, wired to the wrong
 * scenario, or simply broken. `validEvidence` is therefore a runtime contract,
 * not a restatement of `RegionCorrectionEvidence` — every field is re-read
 * through a narrowing helper so each check can actually fail.
 */
const isRecord = (value: unknown): value is Record<string, unknown> => typeof value === "object" && value !== null;
const asString = (value: unknown): string => (typeof value === "string" ? value : "");
const asNumber = (value: unknown): number => (typeof value === "number" ? value : Number.NaN);

const FIDELITIES = ["captured", "composited", "rendered", "sketch"] as const;
const ORIGIN_KINDS = ["served-page", "hub-packet"] as const;

const arrayIncludes = (values: readonly string[], candidate: string): boolean => values.includes(candidate);

function validEvidence(value: unknown): value is ResolvedRegionCorrection {
  if (!isRecord(value)) return false;
  const cropPreviewUrl = value.cropPreviewUrl;
  if (typeof cropPreviewUrl !== "string" || !/^\/(?!\/)[^?#\s]+$/.test(cropPreviewUrl)) return false;
  const evidence = value.evidence;
  if (!isRecord(evidence)) return false;
  if (asString(evidence.schema) !== REGION_CORRECTION_SCHEMA) return false;
  if (asString(evidence.kind) !== "correction") return false;
  if (asString(evidence.storage) !== "local") return false;
  if (evidence.private !== true) return false;
  if (!stableId(asString(evidence.correction_id)) || !stableId(asString(evidence.owner_scenario)) || !stableId(asString(evidence.corpus))) return false;
  const capture = evidence.capture;
  if (!isRecord(capture)) return false;
  const captureID = asString(capture.id);
  if (!stableId(captureID) || !/^sha256:[0-9a-f]{64}$/.test(asString(capture.build_id)) || !/^\/[A-Za-z0-9._~!$&'()*+,;=:@%/-]{0,1023}$/.test(asString(capture.route)) || !stableId(asString(capture.state_id)) || !stableId(asString(capture.data_id))) return false;
  const origin = evidence.origin;
  if (!isRecord(origin)) return false;
  const originKind = asString(origin.kind);
  if (!arrayIncludes(ORIGIN_KINDS, originKind)) return false;
  if (!stableId(asString(origin.id))) return false;
  if (originKind === "served-page" && asString(origin.id) !== captureID) return false;
  if (originKind === "hub-packet" && !stableId(asString(origin.attachment))) return false;
  const image = capture.image;
  if (!isRecord(image)) return false;
  if (!arrayIncludes(FIDELITIES, asString(capture.fidelity)) || !bundlePath(asString(image.path)) || !digest(asString(image.sha256))) return false;
  const viewport = capture.viewport;
  if (!isRecord(viewport)) return false;
  const viewportWidth = asNumber(viewport.width);
  const viewportHeight = asNumber(viewport.height);
  if (!Number.isInteger(viewportWidth) || viewportWidth < 1 || !Number.isInteger(viewportHeight) || viewportHeight < 1) return false;
  const region = evidence.region;
  if (!isRecord(region)) return false;
  const label = asString(region.label);
  if (!stableId(asString(region.id)) || !stableId(asString(region.node_id)) || !label.trim() || label.length > 256 || !stableId(asString(region.component)) || !stableId(asString(region.evidence_id))) return false;
  const crop = region.crop;
  if (!isRecord(crop)) return false;
  if (!bundlePath(asString(crop.path)) || !digest(asString(crop.sha256))) return false;
  const bounds = region.bounds;
  if (!isRecord(bounds)) return false;
  const x = asNumber(bounds.x);
  const y = asNumber(bounds.y);
  const width = asNumber(bounds.width);
  const height = asNumber(bounds.height);
  return [x, y, width, height].every(Number.isFinite) && x >= 0 && y >= 0 && width > 0 && height > 0 && x + width <= 1 && y + height <= 1;
}

export function RegionCorrectionProvider({ children, resolveEvidence, submitCorrection, shortcut = "u", disabled = false }: RegionCorrectionProviderProps) {
  const { t } = useTranslation();
  const [resolvedCorrection, setResolvedCorrection] = useState<ResolvedRegionCorrection | null>(null);
  const evidence = resolvedCorrection?.evidence ?? null;
  const [message, setMessage] = useState("");
  const [status, setStatus] = useState("");
  const [notice, setNotice] = useState("");
  const [selecting, setSelecting] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const point = useRef<Point | null>(null);
  const suppressNextClick = useRef(false);
  const longPressTriggered = useRef(false);
  const removeLayer = useRef<(() => void) | null>(null);
  const [portal, setPortal] = useState<HTMLElement | null>(null);

  useEffect(() => { setPortal(getLayerPortalContainer()); }, []);
  useEffect(() => () => {
    if (timer.current) clearTimeout(timer.current);
    removeLayer.current?.();
  }, []);

  const openFor = useCallback(async (element: HTMLElement) => {
    setStatus("Finding the page evidence…");
    setNotice("");
    try {
      const resolved = await resolveEvidence(element);
      if (!validEvidence(resolved)) {
        setStatus("This part of the page has no complete capture identity yet. No correction was sent.");
        return;
      }
      removeLayer.current?.();
      removeLayer.current = layerManager.push({ id: `region-correction-${resolved.evidence.correction_id}`, kind: "sheet", modal: true, dismiss: () => setResolvedCorrection(null) });
      setResolvedCorrection(resolved);
      setMessage("");
      setStatus("");
      setNotice("");
    } catch {
      setStatus("Page evidence could not be read. No correction was sent.");
    }
  }, [resolveEvidence]);

  const clearTimer = () => {
    if (longPressTriggered.current) {
      suppressNextClick.current = true;
      longPressTriggered.current = false;
    }
    if (timer.current) clearTimeout(timer.current);
    timer.current = null;
    point.current = null;
  };

  const onPointerDown = (event: PointerEvent<HTMLElement>) => {
    if (disabled || event.button !== 0 || event.pointerType === "mouse") return;
    const element = targetElement(event.target);
    if (!element || (isInteractive(element) && !explicitlyMarked(element))) return;
    point.current = { x: event.clientX, y: event.clientY, element };
    timer.current = setTimeout(() => {
      const selected = point.current?.element;
      timer.current = null;
      point.current = null;
      longPressTriggered.current = true;
      if (selected) void openFor(selected);
    }, REGION_LONG_PRESS_MS);
  };

  const onPointerMove = (event: PointerEvent<HTMLElement>) => {
    if (!point.current) return;
    if (Math.hypot(event.clientX - point.current.x, event.clientY - point.current.y) > 10) clearTimer();
  };

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (disabled || event.repeat || !event.altKey || !event.shiftKey || event.key.toLowerCase() !== shortcut.toLowerCase()) return;
      const active = document.activeElement;
      const element = active instanceof HTMLElement ? targetElement(active) : null;
      const fallback = element ?? targetElement(document.elementFromPoint(window.innerWidth / 2, window.innerHeight / 2));
      if (fallback && (!isInteractive(fallback) || explicitlyMarked(fallback))) void openFor(fallback);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [disabled, openFor, shortcut]);

  const close = () => { removeLayer.current?.(); removeLayer.current = null; setResolvedCorrection(null); setStatus(""); };
  const onClick = (event: MouseEvent<HTMLDivElement>) => {
    if (event.target instanceof Element && event.target.closest("[data-ux-provider-ui]")) return;
    if (suppressNextClick.current) {
      suppressNextClick.current = false;
      event.preventDefault();
      event.stopPropagation();
      return;
    }
    if (!selecting) return;
    const target = targetElement(event.target);
    if (!target || (isInteractive(target) && !explicitlyMarked(target))) {
      setStatus("Choose a marked page region. Interactive controls without a marker are skipped to avoid activating them.");
      return;
    }
    event.preventDefault();
    event.stopPropagation();
    setSelecting(false);
    setStatus("");
    void openFor(target);
  };
  const submit = async () => {
    if (!evidence || message.trim().length < 3 || message.trim().length > 500) return;
    setStatus("Sending correction…");
    try {
      await submitCorrection({ ...evidence, text: message.trim() });
      setNotice("Correction queued locally with its page evidence.");
      close();
    } catch {
      setStatus("The correction was not sent. Keep this panel open and try again.");
    }
  };

  // Capture-phase pointer handlers only observe gestures for long-press
  // detection; the wrapper is not an interactive control of its own.
  return (
    // eslint-disable-next-line jsx-a11y/no-static-element-interactions -- pointer capture observes gestures, it does not make the wrapper interactive
    <div
      data-rcl-region-correction-root
      onClickCapture={onClick}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={clearTimer}
      onPointerCancel={clearTimer}
      onContextMenu={(event) => {
        if (point.current) event.preventDefault();
      }}
    >
      {children}
      <StyleSheet libraryId="react-component-library:RegionCorrectionProvider" version="0.1.0-draft.1" css={correctionStyles} />
      {portal &&
        createPortal(
          <div data-ux-provider-ui data-rcl-region-correction-launcher>
            {selecting && (
              <p data-rcl-region-correction-instruction role="status">
                {t("regionCorrection.tapInstruction", {
                  defaultValue: "Tap a marked page region to attach its evidence.",
                })}
              </p>
            )}
            {status && !evidence && (
              <p data-rcl-region-correction-feedback role="status">
                {status}
              </p>
            )}
            {notice && !evidence && (
              <p data-rcl-region-correction-feedback role="status">
                {notice}
              </p>
            )}
            {!evidence && (
              <button
                type="button"
                aria-pressed={selecting}
                aria-keyshortcuts={`Alt+Shift+${shortcut.toUpperCase()}`}
                onClick={() => {
                  setSelecting((value) => !value);
                  setStatus("");
                  setNotice("");
                }}
              >
                {selecting
                  ? t("regionCorrection.cancelSelection", { defaultValue: "Cancel region choice" })
                  : t("regionCorrection.report", { defaultValue: "Report a page issue" })}
              </button>
            )}
          </div>,
          portal,
        )}
      {evidence &&
        portal &&
        createPortal(
          // The backdrop dismisses on press; the panel inside is the
          // interactive surface, so a button role here would be wrong.
          // eslint-disable-next-line jsx-a11y/no-noninteractive-element-interactions -- overlay backdrop dismissal, not element interaction
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby="region-correction-title"
            data-rcl-region-correction-backdrop
            onMouseDown={(event) => {
              if (event.target === event.currentTarget) close();
            }}
          >
            <section data-rcl-region-correction-panel aria-describedby="region-correction-description">
              <div data-rcl-region-correction-header>
                <div>
                  <h2 id="region-correction-title">
                    {t("regionCorrection.title", { defaultValue: "Point out what needs fixing" })}
                  </h2>
                  <p id="region-correction-description">
                    {t("regionCorrection.description", {
                      defaultValue:
                        "This note is queued locally for {{ownerScenario}} with the selected component, route, build, viewport and saved crop.",
                      ownerScenario: evidence.owner_scenario,
                    })}
                  </p>
                </div>
                <button
                  data-rcl-region-correction-close
                  type="button"
                  onClick={close}
                  aria-label={t("regionCorrection.closeLabel", { defaultValue: "Close correction" })}
                >
                  {t("regionCorrection.close", { defaultValue: "Close" })}
                </button>
              </div>
              <figure data-rcl-region-correction-crop>
                <img
                  src={resolvedCorrection?.cropPreviewUrl ?? ""}
                  alt={t("regionCorrection.cropAlt", {
                    defaultValue: "Evidence crop for {{component}}",
                    component: evidence.region.component,
                  })}
                />
                <figcaption>
                  {t("regionCorrection.cropCaption", {
                    defaultValue: "Saved crop · {{evidenceId}}",
                    evidenceId: evidence.region.evidence_id,
                  })}
                </figcaption>
              </figure>
              <dl id="region-correction-context" data-rcl-region-correction-context>
                <dt>{t("regionCorrection.region", { defaultValue: "Region" })}</dt>
                <dd>{evidence.region.id}</dd>
                <dt>{t("regionCorrection.component", { defaultValue: "Component" })}</dt>
                <dd>{evidence.region.component}</dd>
                <dt>{t("regionCorrection.route", { defaultValue: "Route" })}</dt>
                <dd>{evidence.capture.route}</dd>
                <dt>{t("regionCorrection.viewport", { defaultValue: "Viewport" })}</dt>
                <dd>
                  {evidence.capture.viewport.width} × {evidence.capture.viewport.height}
                </dd>
              </dl>
              <label htmlFor="region-correction-text">
                {t("regionCorrection.messageLabel", {
                  defaultValue: "One sentence about the problem",
                })}
              </label>
              <textarea
                id="region-correction-text"
                value={message}
                maxLength={500}
                onChange={(event) => setMessage(event.target.value)}
                placeholder={t("regionCorrection.messagePlaceholder", {
                  defaultValue: "What should change here?",
                })}
              />
              <p data-rcl-region-correction-status role="status">
                {status}
              </p>
              <button
                data-rcl-region-correction-submit
                type="button"
                disabled={message.trim().length < 3 || message.trim().length > 500}
                onClick={() => void submit()}
              >
                {t("regionCorrection.submit", { defaultValue: "Send correction" })}
              </button>
              <p data-rcl-region-correction-shortcut>
                {t("regionCorrection.shortcut", {
                  defaultValue:
                    "Shortcut: Shift + Alt + {{key}} when a page region is focused.",
                  key: shortcut.toUpperCase(),
                })}
              </p>
            </section>
          </div>,
          portal,
        )}
    </div>
  );
}

export default RegionCorrectionProvider;
