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
  /** The last successfully saved source path and digest, used as a cache-buster. */
  sourceRevision?: string;
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
 * A changed version, retry key or saved-source revision is staged in a hidden
 * candidate frame. The last ready frame stays visible until the candidate has
 * passed its readiness and override handshake, so a failed reload cannot blank
 * a specimen that was already on screen.
 */
export function FixtureFrame({
  assetId,
  storyId,
  version,
  override,
  reloadKey = 0,
  sourceRevision = "",
  height = 150,
  theme,
  onStatus,
  onRetry,
  title,
}: FixtureFrameProps) {
  const { t } = useTranslation();
  const activeFrameRef = useRef<HTMLIFrameElement | null>(null);
  const candidateFrameRef = useRef<HTMLIFrameElement | null>(null);
  const onStatusRef = useRef(onStatus);
  onStatusRef.current = onStatus;
  const statusRef = useRef<FixtureStatus>("loading");
  const mounted = useRef(true);
  const readyTimer = useRef<number | undefined>(undefined);
  const [activeUrl, setActiveUrl] = useState("");
  const [probe, setProbe] = useState<"checking" | "ok" | "missing">("checking");
  const [failedUrl, setFailedUrl] = useState<string | null>(null);
  const [status, setStatus] = useState<FixtureStatus>("loading");
  const [message, setMessage] = useState("");
  const [harnessStory, setHarnessStory] = useState("");
  const [harnessVersion, setHarnessVersion] = useState("");
  const activeUrlRef = useRef(activeUrl);
  activeUrlRef.current = activeUrl;
  const overrideRef = useRef(override);
  overrideRef.current = override;

  const url = useMemo(() => {
    const next = new URL(
      `${API_BASE.replace(/\/$/, "")}/preview/${encodeURIComponent(assetId)}/harness.html`,
    );
    next.searchParams.set("v", `${version}:${storyId}:${reloadKey}:${sourceRevision}`);
    next.searchParams.set("story", storyId);
    next.searchParams.set("version", version);
    next.searchParams.set("kit", "vrooli-default");
    return next.toString();
  }, [assetId, reloadKey, sourceRevision, storyId, version]);
  const urlRef = useRef(url);
  urlRef.current = url;

  const transition = useCallback((next: FixtureStatus, detail?: string) => {
    if (!mounted.current) return;
    statusRef.current = next;
    setStatus(next);
    setMessage(detail ?? "");
    onStatusRef.current?.(next, detail);
  }, []);

  const failCandidate = useCallback(
    (failedURL: string, detail: string) => {
      setProbe("missing");
      setFailedUrl(failedURL);
      transition("error", detail);
    },
    [transition],
  );

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      window.clearTimeout(readyTimer.current);
      readyTimer.current = undefined;
    };
  }, []);

  // Probe the next harness before mounting it. Existing ready content remains
  // mounted while the candidate is checked and loaded.
  useEffect(() => {
    const controller = new AbortController();
    let cancelled = false;
    setProbe("checking");
    setFailedUrl(null);
    transition("loading");
    const timer = window.setTimeout(() => controller.abort(), FIXTURE_TIMEOUT_MS);

    fetch(url, { credentials: "same-origin", cache: "force-cache", signal: controller.signal })
      .then((response) => {
        if (cancelled) return;
        window.clearTimeout(timer);
        if (!response.ok) {
          const detail = t("workbench.component.fixtureHttp", {
            defaultValue: "The preview harness returned HTTP {{status}}.",
            status: response.status,
          });
          failCandidate(url, detail);
          return;
        }
        setProbe("ok");
      })
      .catch((error: unknown) => {
        if (cancelled) return;
        window.clearTimeout(timer);
        failCandidate(
          url,
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
  }, [failCandidate, t, transition, url]);

  useEffect(() => {
    const onMessage = (event: MessageEvent) => {
      const isCandidate = event.source === candidateFrameRef.current?.contentWindow;
      const isCurrentActive =
        event.source === activeFrameRef.current?.contentWindow &&
        activeUrlRef.current === urlRef.current;
      if (!isCandidate && !isCurrentActive) return;
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
        if (!overrideRef.current) setActiveUrl(urlRef.current);
      } else if (data.type === "rcl-preview-props-applied") {
        transition("applied");
        setActiveUrl(urlRef.current);
      } else if (data.type === "rcl-preview-props-error" || data.type === "preview-error") {
        const detail =
          data.message ||
          t("workbench.component.fixtureError", { defaultValue: "The specimen did not render." });
        failCandidate(urlRef.current, detail);
      }
    };
    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, [assetId, failCandidate, t, transition]);

  // A harness that loads but never announces itself is also a failure. Timeout
  // the candidate only; the previously committed frame remains visible.
  useEffect(() => {
    if (probe !== "ok" || status !== "loading") return;
    readyTimer.current = window.setTimeout(() => {
      readyTimer.current = undefined;
      if (statusRef.current !== "loading") return;
      failCandidate(
        urlRef.current,
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
  }, [failCandidate, probe, status, t, transition]);

  // The harness exposes a token bridge (`rcl-theme-apply`) and a resolved
  // colour-scheme bridge. Applying both makes the specimen inherit page tokens.
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
    applyTheme(candidateFrameRef.current ?? activeFrameRef.current);
  }, [applyTheme, status, theme?.id, theme?.resolved, tokenSignature]);

  useEffect(() => {
    const frame = candidateFrameRef.current;
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

  // `error` and `load` do not delegate on iframe elements; native listeners
  // are required. Candidate failures are discarded without replacing the
  // active frame, while active-frame failures remain visible for recovery.
  useEffect(() => {
    const frames = [activeFrameRef.current, candidateFrameRef.current].filter(
      (frame, index, all): frame is HTMLIFrameElement =>
        Boolean(frame) && all.indexOf(frame) === index,
    );
    const onLoad = (event: Event) => applyTheme(event.currentTarget as HTMLIFrameElement);
    const onError = (event: Event) => {
      const frame = event.currentTarget as HTMLIFrameElement;
      const detail = t("workbench.component.fixtureIframeError", {
        defaultValue: "The preview harness could not be mounted.",
      });
      if (frame === candidateFrameRef.current && urlRef.current !== activeUrlRef.current) {
        failCandidate(urlRef.current, detail);
      } else {
        transition("error", detail);
      }
    };
    frames.forEach((frame) => {
      frame.addEventListener("load", onLoad);
      frame.addEventListener("error", onError);
    });
    return () => {
      frames.forEach((frame) => {
        frame.removeEventListener("load", onLoad);
        frame.removeEventListener("error", onError);
      });
    };
  }, [activeUrl, applyTheme, failCandidate, probe, t, transition, url]);

  const activeExists = Boolean(activeUrl);
  const frameURLs = activeExists
    ? [activeUrl, ...(probe === "ok" && failedUrl !== url && url !== activeUrl ? [url] : [])]
    : probe === "ok" && failedUrl !== url
      ? [url]
      : [];
  return (
    <span
      style={{ display: "block", height, position: activeExists ? "relative" : undefined }}
      data-fixture-frame=""
      data-fixture-status={status}
    >
      {status === "error" ? (
        <span
          className="wb-fixture-state is-error"
          role="alert"
          style={
            activeExists
              ? {
                  position: "absolute",
                  inset: "auto 6px 6px",
                  zIndex: 1,
                  height: "auto",
                  borderRadius: 6,
                }
              : undefined
          }
        >
          <span>
            {message ||
              t("workbench.component.fixtureError", {
                defaultValue: "The specimen did not render.",
              })}
          </span>
          {onRetry ? (
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
      ) : !activeExists && status === "loading" ? (
        <span className="wb-fixture-state" role="status">
          <span>
            {t("workbench.component.fixtureLoading", { defaultValue: "Loading specimen…" })}
          </span>
        </span>
      ) : null}
      {frameURLs.map((frameURL) => {
        const isActive = frameURL === activeUrl;
        const isCandidate = frameURL === url;
        return (
          <iframe
            key={frameURL}
            ref={(frame) => {
              if (frameURL === activeUrl) activeFrameRef.current = frame;
              if (isCandidate) candidateFrameRef.current = frame;
            }}
            title={title}
            src={frameURL}
            data-testid={isCandidate ? "fixture-iframe" : "fixture-active-iframe"}
            style={{
              width: "100%",
              height: "100%",
              border: 0,
              background: "transparent",
              visibility: isActive ? "visible" : "hidden",
              position: isActive ? "relative" : "absolute",
              inset: 0,
            }}
            sandbox={SANDBOX}
          />
        );
      })}
    </span>
  );
}
