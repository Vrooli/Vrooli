import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { API_BASE } from "../../api/client";
import { useTranslation } from "../../i18n";

export type FixtureStatus = "loading" | "ready" | "error" | "applied";

/** A specimen that has not answered in this long is a failure, not a spinner. */
export const FIXTURE_TIMEOUT_MS = 8_000;

/**
 * The parent ⇄ harness message channel. The harness posts on `window.parent`
 * and the parent only accepts frames it mounted (`event.source`), so these are
 * the only messages that can move the status:
 *
 *   harness → parent  preview-ready · rcl-preview-props-applied
 *                     rcl-preview-props-error · preview-error
 *   parent → harness  rcl-preview-props-override · rcl-preview-props-reset
 *                     rcl-theme-apply · rcl-resolved-theme
 *
 * `allow-same-origin` is required: the harness runs on the API origin so it can
 * import its own modules and read its design tokens, and an opaque sandbox
 * origin fails every same-origin fetch before it can post `preview-ready`.
 * Scripts stay sandboxed and the parent never reads the frame's DOM, so this
 * grants no capability beyond the harness's own document.
 */
const SANDBOX = "allow-scripts allow-same-origin allow-forms";

interface FixtureFrameProps {
  assetId: string;
  storyId: string;
  version: string;
  override?: Record<string, unknown> | null;
  reloadKey?: number;
  height?: number;
  /** Page tokens applied through the harness theme bridge, so the specimen
   * inherits the page's surface, ink and accents instead of a white box. */
  theme?: { id: string; tokens: Record<string, string>; resolved: "light" | "dark" };
  onStatus?: (status: FixtureStatus, message?: string) => void;
  /** Offered next to a failure message so the failure has a next action. */
  onRetry?: () => void;
  title: string;
}

/**
 * The real component harness in an iframe. The workbench never re-implements a
 * specimen: it mounts the same execution target the catalog captures, then
 * uses the documented override protocol to apply prop edits.
 *
 * Loading has a real failure path: the harness URL is probed for an HTTP
 * status before the iframe mounts, the iframe's own error event is handled, and
 * a specimen that never posts `preview-ready` fails after eight seconds. A
 * missing harness therefore reports what happened instead of spinning.
 */
export function FixtureFrame({
  assetId,
  storyId,
  version,
  override,
  reloadKey = 0,
  height = 150,
  theme,
  onStatus,
  onRetry,
  title,
}: FixtureFrameProps) {
  const { t } = useTranslation();
  const frameRef = useRef<HTMLIFrameElement>(null);
  const onStatusRef = useRef(onStatus);
  onStatusRef.current = onStatus;
  const statusRef = useRef<FixtureStatus>("loading");
  const mounted = useRef(true);
  const readyTimer = useRef<number | undefined>(undefined);
  const [status, setStatus] = useState<FixtureStatus>("loading");
  const [message, setMessage] = useState<string>("");
  const [probe, setProbe] = useState<"checking" | "ok" | "missing">("checking");
  const [harnessStory, setHarnessStory] = useState<string>("");
  const [harnessVersion, setHarnessVersion] = useState<string>("");

  const url = useMemo(() => {
    const next = new URL(
      `${API_BASE.replace(/\/$/, "")}/preview/${encodeURIComponent(assetId)}/harness.html`,
    );
    next.searchParams.set("v", `${version}:${storyId}:${reloadKey}`);
    next.searchParams.set("story", storyId);
    next.searchParams.set("version", version);
    next.searchParams.set("kit", "vrooli-default");
    return next.toString();
  }, [assetId, reloadKey, storyId, version]);

  const transition = useCallback((next: FixtureStatus, detail?: string) => {
    if (!mounted.current) return;
    statusRef.current = next;
    setStatus(next);
    setMessage(detail ?? "");
    onStatusRef.current?.(next, detail);
  }, []);

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      window.clearTimeout(readyTimer.current);
      readyTimer.current = undefined;
    };
  }, []);

  // Probe the harness for a real status before mounting it. An iframe does not
  // surface HTTP errors, so without this a 404 looks exactly like a slow load.
  useEffect(() => {
    const controller = new AbortController();
    let cancelled = false;
    setProbe("checking");
    transition("loading");
    const timer = window.setTimeout(() => controller.abort(), FIXTURE_TIMEOUT_MS);

    fetch(url, { credentials: "same-origin", cache: "force-cache", signal: controller.signal })
      .then((response) => {
        if (cancelled) return;
        window.clearTimeout(timer);
        if (!response.ok) {
          setProbe("missing");
          transition(
            "error",
            t("workbench.component.fixtureHttp", {
              defaultValue: "The preview harness returned HTTP {{status}}.",
              status: response.status,
            }),
          );
          return;
        }
        setProbe("ok");
      })
      .catch((error: unknown) => {
        if (cancelled) return;
        window.clearTimeout(timer);
        setProbe("missing");
        transition(
          "error",
          controller.signal.aborted
            ? t("workbench.component.fixtureTimeout", {
                defaultValue: "The preview harness did not answer within {{seconds}} seconds.",
                seconds: FIXTURE_TIMEOUT_MS / 1000,
              })
            : t("workbench.component.fixtureUnreachable", {
                defaultValue: "The preview harness could not be reached: {{message}}",
                message: error instanceof Error ? error.message : String(error),
              }),
        );
      });

    return () => {
      cancelled = true;
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [t, transition, url]);

  useEffect(() => {
    const onMessage = (event: MessageEvent) => {
      if (event.source !== frameRef.current?.contentWindow) return;
      const data = event.data as {
        type?: string;
        id?: string;
        story?: string;
        version?: string;
        message?: string;
      } | null;
      if (!data || data.id !== assetId) return;
      if (data.type === "preview-ready") {
        setHarnessStory(data.story ?? "");
        setHarnessVersion(data.version ?? "");
        transition("ready");
      } else if (data.type === "rcl-preview-props-applied") {
        transition("applied");
      } else if (data.type === "rcl-preview-props-error" || data.type === "preview-error") {
        transition(
          "error",
          data.message ||
            t("workbench.component.fixtureError", { defaultValue: "The specimen did not render." }),
        );
      }
    };
    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, [assetId, t, transition]);

  // A harness that loads but never announces itself is also a failure. The
  // timer lives in a ref so unmount clears it even if the effect is mid-flight.
  useEffect(() => {
    if (probe !== "ok" || status !== "loading") return;
    readyTimer.current = window.setTimeout(() => {
      readyTimer.current = undefined;
      if (statusRef.current !== "loading") return;
      transition(
        "error",
        t("workbench.component.fixtureTimeout", {
          defaultValue: "The preview harness did not answer within {{seconds}} seconds.",
          seconds: FIXTURE_TIMEOUT_MS / 1000,
        }),
      );
    }, FIXTURE_TIMEOUT_MS);
    return () => {
      window.clearTimeout(readyTimer.current);
      readyTimer.current = undefined;
    };
  }, [probe, status, t, transition]);

  // The harness exposes a token bridge (`rcl-theme-apply`) and a resolved
  // colour-scheme bridge. Applying both is what makes the specimen sit on the
  // page's surface rather than its own white document.
  const themeRef = useRef(theme);
  themeRef.current = theme;
  const applyTheme = useCallback((target: HTMLIFrameElement | null) => {
    if (!target?.contentWindow) return;
    const active = themeRef.current;
    if (!active) {
      target.contentWindow.postMessage({ type: "rcl-theme-apply", themeId: "", tokens: {} }, "*");
      return;
    }
    target.contentWindow.postMessage({ type: "rcl-resolved-theme", theme: active.resolved }, "*");
    target.contentWindow.postMessage(
      { type: "rcl-theme-apply", themeId: active.id, tokens: active.tokens },
      "*",
    );
  }, []);

  const tokenSignature = theme ? JSON.stringify(theme.tokens) : "";

  useEffect(() => {
    if (status !== "ready" && status !== "applied") return;
    applyTheme(frameRef.current);
  }, [applyTheme, status, theme?.id, theme?.resolved, tokenSignature]);

  useEffect(() => {
    const frame = frameRef.current;
    if (!frame || status === "loading" || status === "error") return;
    const story = harnessStory || storyId;
    const targetVersion = harnessVersion || version;
    if (override) {
      frame.contentWindow?.postMessage(
        {
          type: "rcl-preview-props-override",
          componentId: assetId,
          story,
          version: targetVersion,
          props: override,
        },
        "*",
      );
    } else {
      frame.contentWindow?.postMessage(
        {
          type: "rcl-preview-props-reset",
          componentId: assetId,
          story,
          version: targetVersion,
        },
        "*",
      );
    }
  }, [assetId, harnessStory, harnessVersion, override, status, storyId, version]);

  // `error` and `load` are not delegated by React on an iframe, so a JSX
  // `onError` never fires. A native listener is the only reliable failure
  // signal; the load listener applies the theme before `preview-ready` lands.
  useEffect(() => {
    const frame = probe === "ok" ? frameRef.current : null;
    if (!frame) return;
    const onLoad = () => applyTheme(frame);
    const onError = () =>
      transition(
        "error",
        t("workbench.component.fixtureIframeError", {
          defaultValue: "The preview harness could not be mounted.",
        }),
      );
    frame.addEventListener("load", onLoad);
    frame.addEventListener("error", onError);
    return () => {
      frame.removeEventListener("load", onLoad);
      frame.removeEventListener("error", onError);
    };
  }, [applyTheme, probe, t, transition]);

  const visible = status === "ready" || status === "applied";

  return (
    <span style={{ display: "block", height }} data-fixture-frame="" data-fixture-status={status}>
      {!visible ? (
        <span
          className={`wb-fixture-state${status === "error" ? " is-error" : ""}`}
          role={status === "error" ? "alert" : "status"}
        >
          <span>
            {status === "error"
              ? message ||
                t("workbench.component.fixtureError", {
                  defaultValue: "The specimen did not render.",
                })
              : t("workbench.component.fixtureLoading", { defaultValue: "Loading specimen…" })}
          </span>
          {status === "error" && onRetry ? (
            <button
              type="button"
              className="wb-btn ghost sm"
              data-testid="fixture-retry"
              onClick={onRetry}
            >
              {t("workbench.component.fixtureRetry", { defaultValue: "Try again" })}
            </button>
          ) : null}
        </span>
      ) : null}
      {probe === "ok" ? (
        <iframe
          ref={frameRef}
          title={title}
          src={url}
          data-testid="fixture-iframe"
          style={{
            width: "100%",
            height: "100%",
            border: 0,
            background: "transparent",
            visibility: visible ? "visible" : "hidden",
          }}
          sandbox={SANDBOX}
        />
      ) : null}
    </span>
  );
}
