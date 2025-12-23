import { ChevronDown, Check, Image as ImageIcon, X } from "lucide-react";
import clsx from "clsx";
import { useEffect, useMemo, useState } from "react";
import { useAssetStore } from "@stores/assetStore";
import {
  BACKGROUND_GROUP_ORDER,
  CURSOR_GROUP_ORDER,
  MAX_BROWSER_SCALE,
  MIN_BROWSER_SCALE,
  MAX_CURSOR_SCALE,
  MIN_CURSOR_SCALE,
  REPLAY_CHROME_OPTIONS,
  REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS,
  REPLAY_CURSOR_POSITIONS,
  DEFAULT_REPLAY_GRADIENT_SPEC,
  buildGradientCss,
  normalizeGradientStops,
  parseGradientCss,
  type ReplayBackgroundImageFit,
  type ReplayGradientSpec,
} from "@/domains/replay-style";
import { AssetPicker } from "@/domains/exports/replay/AssetPicker";
import { RangeSlider } from "@shared/ui";
import type { ReplayCustomizationController } from "./useReplayCustomization";

interface ReplayCustomizationPanelProps {
  controller: ReplayCustomizationController;
}

/**
 * ReplayCustomizationPanel renders the theming + cursor controls for the replay tab.
 * All state is managed by useReplayCustomization; this component is purely presentational.
 */
export function ReplayCustomizationPanel({ controller }: ReplayCustomizationPanelProps) {
  const {
    selectedChromeOption,
    selectedCursorOption,
    selectedCursorPositionOption,
    selectedCursorClickAnimationOption,
    backgroundOptionsByGroup,
    cursorOptionsByGroup,
    replayChromeTheme,
    replayBackgroundTheme,
    replayBackground,
    replayCursorTheme,
    replayCursorInitialPosition,
    replayCursorClickAnimation,
    replayCursorScale,
    replayBrowserScale,
    setReplayChromeTheme,
    setReplayBackground,
    setIsCustomizationCollapsed,
    isCustomizationCollapsed,
    isBackgroundMenuOpen,
    setIsBackgroundMenuOpen,
    isCursorMenuOpen,
    setIsCursorMenuOpen,
    isCursorPositionMenuOpen,
    setIsCursorPositionMenuOpen,
    isCursorClickAnimationMenuOpen,
    setIsCursorClickAnimationMenuOpen,
    backgroundSelectorRef,
    cursorSelectorRef,
    cursorPositionSelectorRef,
    cursorClickAnimationSelectorRef,
    handleBackgroundSelect,
    handleCursorThemeSelect,
    handleCursorPositionSelect,
    handleCursorClickAnimationSelect,
    handleCursorScaleChange,
    handleBrowserScaleChange,
    backgroundLabel,
    backgroundSubtitle,
    backgroundPreviewStyle,
    backgroundPreviewNode,
  } = controller;

  const [imageDraft, setImageDraft] = useState<{
    assetId: string;
    fit: ReplayBackgroundImageFit;
  }>({
    assetId: "",
    fit: "cover",
  });
  const [gradientMode, setGradientMode] = useState<"simple" | "advanced">("simple");
  const [gradientCssDraft, setGradientCssDraft] = useState("");
  const [gradientCssError, setGradientCssError] = useState<string | null>(null);
  const [showBackgroundAssetPicker, setShowBackgroundAssetPicker] = useState(false);
  const { assets } = useAssetStore();

  useEffect(() => {
    if (replayBackground.type !== "image") {
      return;
    }
    setImageDraft({
      assetId: replayBackground.assetId ?? "",
      fit: replayBackground.fit ?? "cover",
    });
  }, [replayBackground]);

  const gradientSpec =
    replayBackground.type === "gradient" ? replayBackground.value : DEFAULT_REPLAY_GRADIENT_SPEC;
  const gradientStops = useMemo(
    () => normalizeGradientStops(gradientSpec.stops),
    [gradientSpec.stops],
  );
  const gradientType = gradientSpec.type ?? "linear";
  const gradientCenter = gradientSpec.center ?? { x: 50, y: 50 };
  const gradientAngle =
    gradientSpec.type === "linear" && typeof gradientSpec.angle === "number"
      ? gradientSpec.angle
      : DEFAULT_REPLAY_GRADIENT_SPEC.angle ?? 135;

  const applyGradientSpec = (next: ReplayGradientSpec) => {
    setReplayBackground({
      type: "gradient",
      value: {
        ...next,
        type: next.type ?? "linear",
        stops: normalizeGradientStops(next.stops),
      },
    });
  };

  const applyImageDraft = (draft: { assetId: string; fit: ReplayBackgroundImageFit }) => {
    setReplayBackground({
      type: "image",
      assetId: draft.assetId.trim() || undefined,
      fit: draft.fit,
    });
  };

  const backgroundMode = replayBackground.type;
  const selectedBackgroundAsset = useMemo(
    () =>
      replayBackground.type === "image" && replayBackground.assetId
        ? assets.find((asset) => asset.id === replayBackground.assetId)
        : null,
    [assets, replayBackground],
  );

  useEffect(() => {
    if (backgroundMode !== "gradient" || gradientMode !== "advanced") {
      return;
    }
    setGradientCssDraft(
      buildGradientCss({
        type: gradientType,
        angle: gradientType === "linear" ? gradientAngle : undefined,
        center: gradientType === "radial" ? gradientCenter : undefined,
        stops: gradientStops,
      }),
    );
    setGradientCssError(null);
  }, [backgroundMode, gradientAngle, gradientCenter, gradientStops, gradientMode, gradientType]);

  const baseGradientSpec: ReplayGradientSpec = {
    type: gradientType,
    angle: gradientType === "linear" ? gradientAngle : undefined,
    center: gradientType === "radial" ? gradientCenter : undefined,
    stops: gradientStops,
  };

  const handleGradientTypeChange = (value: "linear" | "radial") => {
    applyGradientSpec({
      ...baseGradientSpec,
      type: value,
      angle: value === "linear" ? gradientAngle : undefined,
      center: value === "radial" ? gradientCenter : undefined,
    });
  };

  const handleGradientStopChange = (
    index: number,
    update: Partial<{ color: string; position: number }>,
  ) => {
    const nextStops = gradientStops.map((stop, stopIndex) =>
      stopIndex === index
        ? {
            ...stop,
            ...update,
          }
        : stop,
    );
    applyGradientSpec({ ...baseGradientSpec, stops: nextStops });
  };

  const handleAddGradientStop = () => {
    if (gradientStops.length >= 4) {
      return;
    }
    const last = gradientStops[gradientStops.length - 1];
    const prev = gradientStops[gradientStops.length - 2];
    const position = prev && last
      ? Math.round(((prev.position ?? 0) + (last.position ?? 100)) / 2)
      : 50;
    const nextStops = [...gradientStops];
    nextStops.splice(Math.max(1, nextStops.length - 1), 0, {
      color: last?.color ?? "#ffffff",
      position,
    });
    applyGradientSpec({ ...baseGradientSpec, stops: nextStops });
  };

  const handleRemoveGradientStop = (index: number) => {
    if (gradientStops.length <= 2) {
      return;
    }
    const nextStops = gradientStops.filter((_, stopIndex) => stopIndex !== index);
    applyGradientSpec({ ...baseGradientSpec, stops: nextStops });
  };

  const handleApplyGradientCss = () => {
    const parsed = parseGradientCss(gradientCssDraft);
    if (!parsed) {
      setGradientCssError("Enter a valid linear or radial CSS gradient.");
      return;
    }
    setGradientCssError(null);
    applyGradientSpec(parsed);
  };

  return (
    <div className="rounded-2xl border border-white/5 bg-slate-950/40 px-4 py-3 shadow-inner">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">
            Replay customization
          </span>
          <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-slate-500">
            <span>Chrome • {selectedChromeOption.label}</span>
            <span>Background • {backgroundLabel}</span>
            <span>Cursor • {selectedCursorOption.label}</span>
            <span>Click • {selectedCursorClickAnimationOption.label}</span>
          </div>
        </div>
        <button
          type="button"
          onClick={() => setIsCustomizationCollapsed(!isCustomizationCollapsed)}
          className="inline-flex h-8 w-8 items-center justify-center rounded-full border border-white/10 bg-white/5 text-slate-200 transition-colors transition-transform hover:bg-white/10 focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-900"
          aria-expanded={!isCustomizationCollapsed}
          aria-label={
            isCustomizationCollapsed
              ? "Expand replay customization"
              : "Collapse replay customization"
          }
        >
          <ChevronDown
            size={16}
            className={clsx("transition-transform duration-200", {
              "-rotate-180": !isCustomizationCollapsed,
            })}
          />
        </button>
      </div>
      {!isCustomizationCollapsed && (
        <div className="mt-4 space-y-4">
          <div>
            <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
              <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">
                Browser frame
              </span>
              <span className="text-[11px] text-slate-500">
                Customize the replay window
              </span>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              {REPLAY_CHROME_OPTIONS.map((option) => (
                <button
                  key={option.id}
                  type="button"
                  onClick={() => setReplayChromeTheme(option.id)}
                  title={option.subtitle}
                  className={clsx(
                    "rounded-full px-3 py-1.5 text-xs font-medium transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900",
                    replayChromeTheme === option.id
                      ? "bg-flow-accent text-white shadow-[0_12px_35px_rgba(59,130,246,0.45)]"
                      : "bg-slate-900/60 text-slate-300 hover:bg-slate-900/80",
                  )}
                >
                  {option.label}
                </button>
              ))}
            </div>
            <div className="mt-4 rounded-xl border border-white/10 bg-slate-900/60 px-3 py-2">
              <div className="flex items-center justify-between text-[11px] uppercase tracking-[0.24em] text-slate-400">
                <span>Browser size</span>
                <span>{Math.round(replayBrowserScale * 100)}%</span>
              </div>
              <RangeSlider
                min={MIN_BROWSER_SCALE}
                max={MAX_BROWSER_SCALE}
                step={0.05}
                value={replayBrowserScale}
                onChange={handleBrowserScaleChange}
                ariaLabel="Browser size"
                className="mt-2"
              />
              <div className="mt-1 text-[11px] text-slate-500">
                Scale how much of the canvas the browser occupies.
              </div>
            </div>
          </div>

          <div className="border-t border-white/5 pt-3 space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">
              Background
            </span>
            <span className="text-[11px] text-slate-500">
              Set the stage behind the browser
            </span>
          </div>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() =>
                setReplayBackground({
                  type: "theme",
                  id: replayBackgroundTheme,
                })
              }
              className={clsx(
                "rounded-full px-3 py-1.5 text-[11px] font-medium transition-colors",
                backgroundMode === "theme"
                  ? "bg-flow-accent text-white"
                  : "bg-slate-900/60 text-slate-300 hover:bg-slate-900/80",
              )}
            >
              Theme
            </button>
            <button
              type="button"
              onClick={() => applyGradientSpec(baseGradientSpec)}
              className={clsx(
                "rounded-full px-3 py-1.5 text-[11px] font-medium transition-colors",
                backgroundMode === "gradient"
                  ? "bg-flow-accent text-white"
                  : "bg-slate-900/60 text-slate-300 hover:bg-slate-900/80",
              )}
            >
              Gradient
            </button>
            <button
              type="button"
              onClick={() => applyImageDraft(imageDraft)}
              className={clsx(
                "rounded-full px-3 py-1.5 text-[11px] font-medium transition-colors",
                backgroundMode === "image"
                  ? "bg-flow-accent text-white"
                  : "bg-slate-900/60 text-slate-300 hover:bg-slate-900/80",
              )}
            >
              Image
            </button>
          </div>
          <div ref={backgroundSelectorRef} className="relative">
              <button
                type="button"
                className={clsx(
                  "flex w-full items-center justify-between rounded-xl border px-3 py-2 text-left text-sm transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900",
                  isBackgroundMenuOpen
                    ? "border-flow-accent/70 bg-slate-900/80 text-white"
                    : "border-white/10 bg-slate-900/60 text-slate-200 hover:border-flow-accent/40 hover:text-white",
                )}
                onClick={() => {
                  setIsBackgroundMenuOpen(!isBackgroundMenuOpen);
                  setIsCursorMenuOpen(false);
                  setIsCursorPositionMenuOpen(false);
                  setIsCursorClickAnimationMenuOpen(false);
                }}
                aria-haspopup="menu"
                aria-expanded={isBackgroundMenuOpen}
              >
                <span className="flex items-center gap-3">
                  <span
                    aria-hidden
                    className="relative h-10 w-16 overflow-hidden rounded-lg border border-white/10 shadow-inner"
                    style={backgroundPreviewStyle}
                  >
                    {backgroundPreviewNode}
                  </span>
                  <span className="flex flex-col text-xs leading-tight text-slate-300">
                    <span className="text-sm font-medium text-white">
                      {backgroundLabel}
                    </span>
                    <span className="text-[11px] text-slate-400">
                      {backgroundSubtitle}
                    </span>
                  </span>
                </span>
                <ChevronDown
                  size={14}
                  className={clsx(
                    "ml-3 flex-shrink-0 text-slate-400 transition-transform duration-150",
                    isBackgroundMenuOpen ? "rotate-180 text-white" : "",
                  )}
                />
              </button>

              {isBackgroundMenuOpen && (
                <div
                  role="menu"
                  className="absolute right-0 z-30 mt-2 w-full min-w-[260px] rounded-xl border border-white/10 bg-slate-950/95 p-2 shadow-2xl backdrop-blur-md sm:w-80"
                >
                  {BACKGROUND_GROUP_ORDER.map((group) => {
                    const options = backgroundOptionsByGroup[group.id];
                    if (!options || options.length === 0) {
                      return null;
                    }
                    return (
                      <div key={group.id} className="py-1">
                        <div className="px-2 pb-1 text-[10px] uppercase tracking-[0.24em] text-slate-500">
                          {group.label}
                        </div>
                        <div className="space-y-1">
                          {options.map((option) => {
                            const isActive = replayBackgroundTheme === option.id;
                            return (
                              <button
                                key={option.id}
                                type="button"
                                role="menuitemradio"
                                aria-checked={isActive}
                                onClick={() => handleBackgroundSelect(option)}
                                className={clsx(
                                  "flex w-full items-center gap-3 rounded-lg border px-2.5 py-2 text-left transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-950",
                                  isActive
                                    ? "border-flow-accent/80 bg-flow-accent/20 text-white shadow-[0_12px_35px_rgba(59,130,246,0.32)]"
                                    : "border-white/5 bg-slate-900/60 text-slate-300 hover:border-flow-accent/40 hover:text-white",
                                )}
                              >
                                <span
                                  className="relative h-10 w-16 flex-shrink-0 overflow-hidden rounded-lg border border-white/10 shadow-inner"
                                  style={option.previewStyle}
                                >
                                  {option.previewNode}
                                </span>
                                <span className="flex flex-1 flex-col text-xs text-slate-300">
                                  <span className="flex items-center justify-between gap-2 text-sm font-medium">
                                    <span>{option.label}</span>
                                    {isActive && <Check size={14} className="text-flow-accent" />}
                                  </span>
                                  <span className="text-[11px] text-slate-400">
                                    {option.subtitle}
                                  </span>
                                </span>
                              </button>
                            );
                          })}
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </div>

          {backgroundMode === "gradient" && (
            <div className="rounded-xl border border-white/10 bg-slate-900/60 px-3 py-3 space-y-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">
                  Gradient editor
                </span>
                <div className="flex items-center gap-2 text-[11px]">
                  <button
                    type="button"
                    onClick={() => setGradientMode("simple")}
                    className={clsx(
                      "rounded-full px-3 py-1.5 font-medium transition-colors",
                      gradientMode === "simple"
                        ? "bg-flow-accent text-white"
                        : "bg-slate-900/60 text-slate-300 hover:bg-slate-900/80",
                    )}
                  >
                    Simple
                  </button>
                  <button
                    type="button"
                    onClick={() => setGradientMode("advanced")}
                    className={clsx(
                      "rounded-full px-3 py-1.5 font-medium transition-colors",
                      gradientMode === "advanced"
                        ? "bg-flow-accent text-white"
                        : "bg-slate-900/60 text-slate-300 hover:bg-slate-900/80",
                    )}
                  >
                    Advanced
                  </button>
                </div>
              </div>

              {gradientMode === "simple" ? (
                <div className="space-y-3">
                  <div className="flex flex-wrap gap-2">
                    <button
                      type="button"
                      onClick={() => handleGradientTypeChange("linear")}
                      className={clsx(
                        "rounded-full px-3 py-1.5 text-[11px] font-medium transition-colors",
                        gradientType === "linear"
                          ? "bg-flow-accent text-white"
                          : "bg-slate-900/60 text-slate-300 hover:bg-slate-900/80",
                      )}
                    >
                      Linear
                    </button>
                    <button
                      type="button"
                      onClick={() => handleGradientTypeChange("radial")}
                      className={clsx(
                        "rounded-full px-3 py-1.5 text-[11px] font-medium transition-colors",
                        gradientType === "radial"
                          ? "bg-flow-accent text-white"
                          : "bg-slate-900/60 text-slate-300 hover:bg-slate-900/80",
                      )}
                    >
                      Radial
                    </button>
                  </div>

                  {gradientType === "linear" ? (
                    <div>
                      <div className="flex items-center justify-between text-[11px] uppercase tracking-[0.24em] text-slate-400">
                        <span>Gradient angle</span>
                        <span>{Math.round(gradientAngle)}°</span>
                      </div>
                      <RangeSlider
                        min={0}
                        max={360}
                        step={5}
                        value={gradientAngle}
                        onChange={(value) =>
                          applyGradientSpec({
                            ...baseGradientSpec,
                            type: "linear",
                            angle: value,
                            center: undefined,
                          })
                        }
                        ariaLabel="Gradient angle"
                        className="mt-2"
                      />
                    </div>
                  ) : (
                    <div className="space-y-2">
                      <div className="flex items-center justify-between text-[11px] uppercase tracking-[0.24em] text-slate-400">
                        <span>Gradient center</span>
                        <span>
                          {Math.round(gradientCenter.x)}% / {Math.round(gradientCenter.y)}%
                        </span>
                      </div>
                      <div className="grid grid-cols-2 gap-2">
                        <label className="text-[11px] text-slate-400 flex flex-col gap-1">
                          X position
                          <RangeSlider
                            min={0}
                            max={100}
                            step={1}
                            value={gradientCenter.x}
                            onChange={(value) =>
                              applyGradientSpec({
                                ...baseGradientSpec,
                                type: "radial",
                                center: { ...gradientCenter, x: value },
                              })
                            }
                            ariaLabel="Radial center X"
                          />
                        </label>
                        <label className="text-[11px] text-slate-400 flex flex-col gap-1">
                          Y position
                          <RangeSlider
                            min={0}
                            max={100}
                            step={1}
                            value={gradientCenter.y}
                            onChange={(value) =>
                              applyGradientSpec({
                                ...baseGradientSpec,
                                type: "radial",
                                center: { ...gradientCenter, y: value },
                              })
                            }
                            ariaLabel="Radial center Y"
                          />
                        </label>
                      </div>
                    </div>
                  )}

                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-[11px] uppercase tracking-[0.24em] text-slate-400">
                      <span>Stops</span>
                      <span>{gradientStops.length} / 4</span>
                    </div>
                    <div className="space-y-2">
                      {gradientStops.map((stop, index) => (
                        <div key={`gradient-stop-${index}`} className="flex items-center gap-2">
                          <input
                            type="color"
                            value={stop.color}
                            onChange={(event) =>
                              handleGradientStopChange(index, { color: event.target.value })
                            }
                            className="h-8 w-10 rounded-lg border border-white/10 bg-slate-900/70 p-1"
                          />
                          <RangeSlider
                            min={0}
                            max={100}
                            step={1}
                            value={stop.position ?? 0}
                            onChange={(value) =>
                              handleGradientStopChange(index, { position: value })
                            }
                            ariaLabel={`Gradient stop ${index + 1} position`}
                            className="flex-1"
                          />
                          <span className="w-10 text-right text-[11px] text-slate-400">
                            {Math.round(stop.position ?? 0)}%
                          </span>
                          {gradientStops.length > 2 && (
                            <button
                              type="button"
                              onClick={() => handleRemoveGradientStop(index)}
                              className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-white/10 text-slate-400 hover:text-white"
                              aria-label="Remove gradient stop"
                            >
                              <X size={12} />
                            </button>
                          )}
                        </div>
                      ))}
                    </div>
                    <button
                      type="button"
                      onClick={handleAddGradientStop}
                      disabled={gradientStops.length >= 4}
                      className="w-full rounded-lg border border-white/10 bg-slate-900/60 px-3 py-2 text-[11px] text-slate-300 hover:bg-slate-900/80 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      Add stop
                    </button>
                  </div>
                </div>
              ) : (
                <div className="space-y-2">
                  <label className="text-[11px] text-slate-400">CSS gradient</label>
                  <textarea
                    value={gradientCssDraft}
                    onChange={(event) => setGradientCssDraft(event.target.value)}
                    rows={3}
                    className="w-full rounded-lg border border-white/10 bg-slate-900/70 px-2.5 py-2 text-xs text-slate-100 focus:outline-none focus:ring-2 focus:ring-flow-accent/50"
                    placeholder="linear-gradient(135deg, #38bdf8 0%, #818cf8 100%)"
                  />
                  {gradientCssError && (
                    <p className="text-[11px] text-rose-300">{gradientCssError}</p>
                  )}
                  <button
                    type="button"
                    onClick={handleApplyGradientCss}
                    className="w-full rounded-lg bg-flow-accent px-3 py-2 text-[11px] font-medium text-white shadow-[0_12px_35px_rgba(59,130,246,0.35)]"
                  >
                    Apply CSS
                  </button>
                  <p className="text-[11px] text-slate-500">
                    Supports linear-gradient and radial-gradient syntax.
                  </p>
                </div>
              )}
            </div>
          )}

          {backgroundMode === "image" && (
            <div className="rounded-xl border border-white/10 bg-slate-900/60 px-3 py-3 space-y-2">
              <div className="space-y-2">
                <label className="text-[11px] text-slate-400">Background image asset</label>
                {selectedBackgroundAsset ? (
                  <div className="flex items-center gap-3 rounded-lg border border-white/10 bg-slate-900/70 p-3">
                    <div className="h-11 w-11 overflow-hidden rounded-lg bg-slate-950">
                      {selectedBackgroundAsset.thumbnail ? (
                        <img
                          src={selectedBackgroundAsset.thumbnail}
                          alt={selectedBackgroundAsset.name}
                          className="h-full w-full object-cover"
                        />
                      ) : (
                        <div className="flex h-full w-full items-center justify-center">
                          <ImageIcon size={18} className="text-slate-500" />
                        </div>
                      )}
                    </div>
                    <div className="min-w-0 flex-1">
                      <span className="block truncate text-xs text-white">{selectedBackgroundAsset.name}</span>
                      <span className="text-[11px] text-slate-500">
                        {selectedBackgroundAsset.width}x{selectedBackgroundAsset.height}
                      </span>
                    </div>
                    <div className="flex gap-2">
                      <button
                        type="button"
                        onClick={() => setShowBackgroundAssetPicker(true)}
                        className="rounded-lg px-2.5 py-1 text-[11px] text-flow-accent transition-colors hover:bg-blue-900/30"
                      >
                        Change
                      </button>
                      <button
                        type="button"
                        onClick={() => {
                          const next = { ...imageDraft, assetId: "" };
                          setImageDraft(next);
                          applyImageDraft(next);
                        }}
                        className="rounded-lg p-1.5 text-slate-400 transition-colors hover:bg-red-900/20 hover:text-red-300"
                      >
                        <X size={12} />
                      </button>
                    </div>
                  </div>
                ) : (
                  <button
                    type="button"
                    onClick={() => setShowBackgroundAssetPicker(true)}
                    className="flex w-full items-center justify-center gap-2 rounded-lg border-2 border-dashed border-white/10 bg-slate-900/70 px-3 py-3 text-[11px] text-slate-300 transition-colors hover:border-flow-accent/50 hover:text-white"
                  >
                    <ImageIcon size={16} />
                    Select background asset
                  </button>
                )}
              </div>
              <label className="text-[11px] text-slate-400 flex flex-col gap-1">
                Image fit
                <select
                  value={imageDraft.fit}
                  onChange={(event) => {
                    const next = { ...imageDraft, fit: event.target.value as ReplayBackgroundImageFit };
                    setImageDraft(next);
                    applyImageDraft(next);
                  }}
                  className="w-full rounded-lg border border-white/10 bg-slate-900/70 px-2.5 py-2 text-xs text-slate-100 focus:outline-none focus:ring-2 focus:ring-flow-accent/50"
                >
                  <option value="cover">Cover</option>
                  <option value="contain">Contain</option>
                </select>
              </label>
              <p className="text-[11px] text-slate-500">
                Choose a background asset from your brand library.
              </p>
            </div>
          )}

          <div className="border-t border-white/5 pt-3 space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">
                Cursor
              </span>
              <span className="text-[11px] text-slate-500">
                Style the virtual pointer overlay
              </span>
            </div>
            <div className="flex flex-col gap-3 lg:flex-row lg:flex-wrap lg:items-start lg:justify-between">
              <div ref={cursorSelectorRef} className="relative flex-1 min-w-[220px]">
                <button
                  type="button"
                  className={clsx(
                    "flex w-full items-center justify-between rounded-xl border px-3 py-2 text-left text-sm transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900",
                    isCursorMenuOpen
                      ? "border-flow-accent/70 bg-slate-900/80 text-white"
                      : "border-white/10 bg-slate-900/60 text-slate-200 hover:border-flow-accent/40 hover:text-white",
                  )}
                  onClick={() => {
                    setIsCursorMenuOpen(!isCursorMenuOpen);
                    setIsBackgroundMenuOpen(false);
                    setIsCursorPositionMenuOpen(false);
                    setIsCursorClickAnimationMenuOpen(false);
                  }}
                  aria-haspopup="menu"
                  aria-expanded={isCursorMenuOpen}
                >
                  <span className="flex items-center gap-3">
                    <span className="relative flex h-10 w-12 items-center justify-center overflow-hidden rounded-lg border border-white/10 bg-slate-900/60">
                      {selectedCursorOption.preview}
                    </span>
                    <span className="flex flex-col text-xs leading-tight text-slate-300">
                      <span className="text-sm font-medium text-white">
                        {selectedCursorOption.label}
                      </span>
                      <span className="text-[11px] text-slate-400">
                        {selectedCursorOption.subtitle}
                      </span>
                    </span>
                  </span>
                  <ChevronDown
                    size={14}
                    className={clsx(
                      "ml-3 flex-shrink-0 text-slate-400 transition-transform duration-150",
                      isCursorMenuOpen ? "rotate-180 text-white" : "",
                    )}
                  />
                </button>
                {isCursorMenuOpen && (
                  <div
                    role="menu"
                    className="absolute right-0 z-30 mt-2 w-full min-w-[240px] rounded-xl border border-white/10 bg-slate-950/95 p-2 shadow-[0_20px_50px_rgba(15,23,42,0.55)] backdrop-blur"
                  >
                    {CURSOR_GROUP_ORDER.map((group) => {
                      const options = cursorOptionsByGroup[group.id];
                      if (!options || options.length === 0) {
                        return null;
                      }
                      return (
                        <div key={group.id} className="mb-2 last:mb-0">
                          <div className="px-2 pb-1 text-[10px] uppercase tracking-[0.24em] text-slate-500">
                            {group.label}
                          </div>
                          <div className="space-y-1">
                            {options.map((option) => {
                              const isActive = replayCursorTheme === option.id;
                              return (
                                <button
                                  key={option.id}
                                  type="button"
                                  role="menuitemradio"
                                  aria-checked={isActive}
                                  onClick={() => handleCursorThemeSelect(option.id)}
                                  className={clsx(
                                    "flex w-full items-center gap-3 rounded-lg border px-2.5 py-2 text-left transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-950",
                                    isActive
                                      ? "border-flow-accent/80 bg-flow-accent/20 text-white shadow-[0_12px_35px_rgba(59,130,246,0.32)]"
                                      : "border-white/5 bg-slate-900/60 text-slate-300 hover:border-flow-accent/40 hover:text-white",
                                  )}
                                >
                                  <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-slate-900/70">
                                    {option.preview}
                                  </span>
                                  <span className="flex flex-1 flex-col text-xs text-slate-300">
                                    <span className="flex items-center justify-between gap-2 text-sm font-medium">
                                      <span>{option.label}</span>
                                      {isActive && <Check size={14} className="text-flow-accent" />}
                                    </span>
                                    <span className="text-[11px] text-slate-400">
                                      {option.subtitle}
                                    </span>
                                  </span>
                                </button>
                              );
                            })}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
              <div ref={cursorPositionSelectorRef} className="relative flex flex-1 flex-col gap-2 lg:max-w-xs">
                <span className="text-[10px] uppercase tracking-[0.24em] text-slate-500">
                  Initial placement
                </span>
                <button
                  type="button"
                  className={clsx(
                    "flex w-full items-center justify-between rounded-xl border px-3 py-2 text-left text-sm transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900",
                    isCursorPositionMenuOpen
                      ? "border-flow-accent/70 bg-slate-900/80 text-white"
                      : "border-white/10 bg-slate-900/60 text-slate-200 hover:border-flow-accent/40 hover:text-white",
                  )}
                  onClick={() => {
                    setIsCursorPositionMenuOpen(!isCursorPositionMenuOpen);
                    setIsBackgroundMenuOpen(false);
                    setIsCursorMenuOpen(false);
                    setIsCursorClickAnimationMenuOpen(false);
                  }}
                  aria-haspopup="menu"
                  aria-expanded={isCursorPositionMenuOpen}
                >
                  <span className="flex flex-col text-xs leading-tight text-slate-300">
                    <span className="text-sm font-medium text-white">
                      {selectedCursorPositionOption.label}
                    </span>
                    <span className="text-[11px] text-slate-400">
                      {selectedCursorPositionOption.subtitle}
                    </span>
                  </span>
                  <ChevronDown
                    size={14}
                    className={clsx(
                      "ml-3 flex-shrink-0 text-slate-400 transition-transform duration-150",
                      isCursorPositionMenuOpen ? "rotate-180 text-white" : "",
                    )}
                  />
                </button>
                {isCursorPositionMenuOpen && (
                  <div
                    role="menu"
                    className="absolute right-0 z-30 mt-2 w-full rounded-xl border border-white/10 bg-slate-950/95 p-2 shadow-[0_18px_45px_rgba(15,23,42,0.55)] backdrop-blur"
                  >
                    <div className="space-y-1">
                      {REPLAY_CURSOR_POSITIONS.map((option) => {
                        const isActive = replayCursorInitialPosition === option.id;
                        return (
                          <button
                            key={option.id}
                            type="button"
                            role="menuitemradio"
                            aria-checked={isActive}
                            onClick={() => handleCursorPositionSelect(option.id)}
                            className={clsx(
                              "w-full rounded-lg border px-2.5 py-2 text-left text-xs transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-950",
                              isActive
                                ? "border-flow-accent/80 bg-flow-accent/20 text-white shadow-[0_12px_35px_rgba(59,130,246,0.32)]"
                                : "border-white/5 bg-slate-900/60 text-slate-300 hover:border-flow-accent/40 hover:text-white",
                            )}
                          >
                            <span className="flex flex-col text-slate-300">
                              <span className="flex items-center justify-between gap-2 text-sm font-medium">
                                <span>{option.label}</span>
                                {isActive && <Check size={14} className="text-flow-accent" />}
                              </span>
                              <span className="text-[11px] text-slate-400">
                                {option.subtitle}
                              </span>
                            </span>
                          </button>
                        );
                      })}
                    </div>
                  </div>
                )}
              </div>
              <div ref={cursorClickAnimationSelectorRef} className="relative flex flex-1 flex-col gap-2 lg:max-w-xs">
                <span className="text-[10px] uppercase tracking-[0.24em] text-slate-500">
                  Click animation
                </span>
                <button
                  type="button"
                  className={clsx(
                    "flex w-full items-center justify-between rounded-xl border px-3 py-2 text-left text-sm transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900",
                    isCursorClickAnimationMenuOpen
                      ? "border-flow-accent/70 bg-slate-900/80 text-white"
                      : "border-white/10 bg-slate-900/60 text-slate-200 hover:border-flow-accent/40 hover:text-white",
                  )}
                  onClick={() => {
                    setIsCursorClickAnimationMenuOpen(!isCursorClickAnimationMenuOpen);
                    setIsBackgroundMenuOpen(false);
                    setIsCursorMenuOpen(false);
                    setIsCursorPositionMenuOpen(false);
                  }}
                  aria-haspopup="menu"
                  aria-expanded={isCursorClickAnimationMenuOpen}
                >
                  <span className="flex flex-col text-xs leading-tight text-slate-300">
                    <span className="text-sm font-medium text-white">
                      {selectedCursorClickAnimationOption.label}
                    </span>
                    <span className="text-[11px] text-slate-400">
                      {selectedCursorClickAnimationOption.subtitle}
                    </span>
                  </span>
                  <ChevronDown
                    size={14}
                    className={clsx(
                      "ml-3 flex-shrink-0 text-slate-400 transition-transform duration-150",
                      isCursorClickAnimationMenuOpen ? "rotate-180 text-white" : "",
                    )}
                  />
                </button>
                {isCursorClickAnimationMenuOpen && (
                  <div
                    role="menu"
                    className="absolute right-0 z-30 mt-2 w-full rounded-xl border border-white/10 bg-slate-950/95 p-2 shadow-[0_18px_45px_rgba(15,23,42,0.55)] backdrop-blur"
                  >
                    <div className="space-y-1">
                      {REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS.map((option) => {
                        const isActive = replayCursorClickAnimation === option.id;
                        return (
                          <button
                            key={option.id}
                            type="button"
                            role="menuitemradio"
                            aria-checked={isActive}
                            onClick={() => handleCursorClickAnimationSelect(option.id)}
                            className={clsx(
                              "w-full rounded-lg border px-2.5 py-2 text-left text-xs transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-950",
                              isActive
                                ? "border-flow-accent/80 bg-flow-accent/20 text-white shadow-[0_12px_35px_rgba(59,130,246,0.32)]"
                                : "border-white/5 bg-slate-900/60 text-slate-300 hover:border-flow-accent/40 hover:text-white",
                            )}
                          >
                            <span className="flex flex-col text-slate-300">
                              <span className="flex items-center justify-between gap-2 text-sm font-medium">
                                <span>{option.label}</span>
                                {isActive && <Check size={14} className="text-flow-accent" />}
                              </span>
                              <span className="text-[11px] text-slate-400">
                                {option.subtitle}
                              </span>
                            </span>
                          </button>
                        );
                      })}
                    </div>
                  </div>
                )}
              </div>
              <div className="flex flex-1 flex-col gap-2 lg:max-w-[260px]">
                <span className="text-[10px] uppercase tracking-[0.24em] text-slate-500">
                  Cursor scale
                </span>
                <input
                  type="range"
                  min={MIN_CURSOR_SCALE}
                  max={MAX_CURSOR_SCALE}
                  step={0.05}
                  value={replayCursorScale}
                  onChange={(event) => handleCursorScaleChange(Number(event.target.value))}
                  className="w-full accent-flow-accent"
                />
                <div className="flex items-center justify-between text-xs text-slate-400">
                  <span>Precision</span>
                  <span>Emphasis</span>
                </div>
                <div className="rounded-lg border border-white/10 bg-slate-900/60 px-3 py-2 text-sm text-slate-200">
                  Scale: <span className="font-semibold text-white">{Math.round(replayCursorScale * 100)}%</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      <AssetPicker
        isOpen={showBackgroundAssetPicker}
        onClose={() => setShowBackgroundAssetPicker(false)}
        onSelect={(assetId) => {
          const next = { ...imageDraft, assetId: assetId ?? "" };
          setImageDraft(next);
          applyImageDraft(next);
        }}
        selectedId={imageDraft.assetId || null}
        filterType="background"
        title="Select Background Image"
      />
    </div>
  );
}

export default ReplayCustomizationPanel;
