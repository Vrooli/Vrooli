import { useEffect, useRef, useState } from "react";

import { API_BASE } from "../../api/client";
import { useTranslation } from "../../i18n";

export type FixtureStatus = "loading" | "ready" | "error" | "applied";

interface FixtureFrameProps {
  assetId: string;
  storyId: string;
  version: string;
  override?: Record<string, unknown> | null;
  reloadKey?: number;
  height?: number;
  onStatus?: (status: FixtureStatus, message?: string) => void;
  title: string;
}

/**
 * The real component harness in an iframe. The workbench never re-implements a
 * specimen: it mounts the same execution target the catalog captures, then
 * uses the documented override protocol to apply prop edits.
 */
export function FixtureFrame({
  assetId,
  storyId,
  version,
  override,
  reloadKey = 0,
  height = 150,
  onStatus,
  title,
}: FixtureFrameProps) {
  const { t } = useTranslation();
  const frameRef = useRef<HTMLIFrameElement>(null);
  const onStatusRef = useRef(onStatus);
  onStatusRef.current = onStatus;
  const [status, setStatus] = useState<FixtureStatus>("loading");
  const [message, setMessage] = useState<string>("");
  const [harnessStory, setHarnessStory] = useState<string>("");
  const [harnessVersion, setHarnessVersion] = useState<string>("");

  const url = new URL(
    `${API_BASE.replace(/\/$/, "")}/preview/${encodeURIComponent(assetId)}/harness.html`,
  );
  url.searchParams.set("v", `${version}:${storyId}:${reloadKey}`);
  url.searchParams.set("story", storyId);
  url.searchParams.set("version", version);
  url.searchParams.set("kit", "vrooli-default");

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
        setStatus("ready");
        setMessage("");
        onStatusRef.current?.("ready");
      } else if (data.type === "rcl-preview-props-applied") {
        setStatus("applied");
        setMessage("");
        onStatusRef.current?.("applied");
      } else if (data.type === "rcl-preview-props-error" || data.type === "preview-error") {
        setStatus("error");
        setMessage(data.message ?? "");
        onStatusRef.current?.("error", data.message);
      }
    };
    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, [assetId]);

  useEffect(() => {
    setStatus("loading");
    setMessage("");
  }, [assetId, storyId, version, reloadKey]);

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

  return (
    <span style={{ display: "block", height }} data-fixture-frame="" data-fixture-status={status}>
      {status === "loading" ? (
        <span className="wb-fixture-state">
          {t("workbench.component.fixtureLoading", { defaultValue: "Loading specimen…" })}
        </span>
      ) : null}
      {status === "error" ? (
        <span className="wb-fixture-state" role="alert">
          {message ||
            t("workbench.component.fixtureError", { defaultValue: "The specimen did not render." })}
        </span>
      ) : null}
      <iframe
        ref={frameRef}
        title={title}
        src={url.toString()}
        style={{
          width: "100%",
          height: "100%",
          border: 0,
          background: "#fff",
          visibility: status === "loading" || status === "error" ? "hidden" : "visible",
        }}
        sandbox="allow-scripts allow-same-origin allow-forms"
      />
    </span>
  );
}
