import { useLayoutEffect, useRef, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { ArrowLeft, Copy, Monitor, Smartphone, Tablet } from "lucide-react";

import { useTranslation } from "../../i18n";
import { assetVersion, useComponentBundle } from "../data";
import { DeviceFrame } from "../component/DeviceFrame";
import { FixtureFrame } from "../component/FixtureFrame";
import { presetFor, type DeviceKind } from "../component/viewports";
import { Chip, Segmented, Select, Skeleton } from "../primitives";

type Device = DeviceKind;

export function PreviewPage() {
  const { t } = useTranslation();
  const { id = "" } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const bundle = useComponentBundle(id);
  const [device, setDevice] = useState<Device>(() => {
    const requested = searchParams.get("device");
    if (requested === "desktop" || requested === "tablet" || requested === "phone")
      return requested;
    // Opening a share link on a phone should show the phone composition, not a
    // 1180px frame clamped into a narrow column.
    if (typeof window !== "undefined" && window.matchMedia("(max-width: 640px)").matches) {
      return "phone";
    }
    return "desktop";
  });
  const [copied, setCopied] = useState(false);
  const stageRef = useRef<HTMLElement>(null);
  const preset = presetFor(device);
  const [scale, setScale] = useState(1);

  useLayoutEffect(() => {
    const stage = stageRef.current;
    if (!stage) return;
    const fit = () => {
      const available = stage.clientWidth - 48;
      const next = Math.max(0.28, Math.min(1, available / Math.max(280, preset.width)));
      setScale((current) => (Math.abs(current - next) < 0.005 ? current : next));
    };
    fit();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(fit);
    observer.observe(stage);
    return () => observer.disconnect();
  }, [preset.width]);

  const stories = bundle.data?.stories ?? [];
  const selectedStoryId = searchParams.get("story") ?? stories[0]?.id ?? "";
  const selectedStory = stories.find((story) => story.id === selectedStoryId) ?? stories[0];

  const share = async () => {
    try {
      await navigator.clipboard.writeText(window.location.href);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      setCopied(false);
    }
  };

  return (
    <div className="wb-root" data-workbench="nocturne" data-testid="preview-popout">
      <div className="wb-preview-page">
        <header className="wb-preview-bar">
          <h1 className="rcl-visually-hidden">
            {bundle.data?.asset.displayName ??
              t("workbench.preview.heading", { defaultValue: "Component preview" })}
          </h1>
          <div className="wb-row" style={{ minWidth: 0 }}>
            <Link
              to={`/assets/${encodeURIComponent(id)}`}
              className="wb-iconbtn"
              aria-label={t("workbench.preview.back", { defaultValue: "Back to asset" })}
            >
              <ArrowLeft aria-hidden size={15} />
            </Link>
            <span style={{ minWidth: 0 }}>
              <b
                style={{
                  display: "block",
                  overflow: "hidden",
                  fontSize: 13,
                  textOverflow: "ellipsis",
                  whiteSpace: "nowrap",
                }}
              >
                {bundle.data?.asset.displayName ??
                  t("workbench.preview.loading", { defaultValue: "Loading…" })}
              </b>
              <small className="wb-mono wb-muted-2">
                {bundle.data?.asset
                  ? `${assetVersion(bundle.data.asset)} · ${selectedStory?.name ?? "story"}`
                  : ""}
              </small>
            </span>
          </div>
          <div className="wb-row" style={{ flexWrap: "wrap" }}>
            <Select
              aria-label={t("workbench.preview.story", { defaultValue: "Story" })}
              value={selectedStoryId}
              onChange={(event) => {
                const params = new URLSearchParams(searchParams);
                params.set("story", event.target.value);
                setSearchParams(params, { replace: true });
              }}
              style={{ width: "auto", minWidth: 160 }}
            >
              {stories.map((story) => (
                <option key={story.id} value={story.id}>
                  {story.name || story.id}
                </option>
              ))}
            </Select>
            <Segmented
              label={t("workbench.preview.device", { defaultValue: "Device" })}
              value={device}
              onChange={(next) => {
                setDevice(next);
                const params = new URLSearchParams(searchParams);
                params.set("device", next);
                setSearchParams(params, { replace: true });
              }}
              items={[
                {
                  id: "desktop",
                  label: <Monitor aria-hidden size={13} />,
                  ariaLabel: t("workbench.preview.desktop", { defaultValue: "Desktop" }),
                },
                {
                  id: "tablet",
                  label: <Tablet aria-hidden size={13} />,
                  ariaLabel: t("workbench.preview.tablet", { defaultValue: "Tablet" }),
                },
                {
                  id: "phone",
                  label: <Smartphone aria-hidden size={13} />,
                  ariaLabel: t("workbench.preview.phone", { defaultValue: "Phone" }),
                },
              ]}
            />
            <Chip tone={bundle.data?.experience?.evidenceStatus === "fresh" ? "green" : "amber"}>
              {bundle.data?.experience?.evidenceStatus ?? "unmeasured"}
            </Chip>
            <button type="button" className="wb-btn ghost sm" onClick={() => void share()}>
              <Copy aria-hidden size={13} />
              {copied
                ? t("workbench.preview.copied", { defaultValue: "Copied" })
                : t("workbench.preview.share", { defaultValue: "Share" })}
            </button>
          </div>
        </header>

        <main className="wb-preview-stage" ref={stageRef}>
          {bundle.isLoading ? (
            <Skeleton height={480} width="min(1180px, 90vw)" />
          ) : bundle.isError || !bundle.data ? (
            <div className="wb-banner is-red" role="alert">
              {t("workbench.preview.error", {
                defaultValue: "The asset could not be loaded. Return to the component and retry.",
              })}
            </div>
          ) : selectedStory ? (
            <DeviceFrame
              device={device}
              width={preset.width}
              height={preset.height}
              scale={scale}
              label={t("workbench.preview.frameLabel", {
                defaultValue: "{{name}} at {{width}} pixels in a {{device}} frame",
                name: selectedStory.name || selectedStory.id,
                width: preset.width,
                device,
              })}
              url={`preview/${bundle.data.asset.libraryId}`}
            >
              <div style={{ position: "absolute", inset: 0 }}>
                <FixtureFrame
                  assetId={id}
                  storyId={selectedStory.id}
                  version={assetVersion(bundle.data.asset)}
                  height={preset.height}
                  title={t("workbench.preview.title", {
                    defaultValue: "{{name}} preview",
                    name: selectedStory.name || selectedStory.id,
                  })}
                />
              </div>
            </DeviceFrame>
          ) : (
            <div className="wb-empty">
              <div className="wb-empty-title">
                {t("workbench.preview.noStories", { defaultValue: "No story to preview" })}
              </div>
              <p>
                {t("workbench.preview.noStoriesBody", {
                  defaultValue: "This asset has no declared story.",
                })}
              </p>
            </div>
          )}
        </main>
      </div>
    </div>
  );
}
