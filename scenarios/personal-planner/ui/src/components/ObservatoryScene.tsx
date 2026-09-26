import type { CSSProperties, ReactNode } from "react";
import { useBreakpoint } from "../hooks/useBreakpoint";
import { useTheme } from "../theme/ThemeProvider";
import { useSampledSceneColors } from "../theme/observatoryAppearance";

type SceneKind = "plan" | "focus" | "settings" | "goals" | "review";

/** Shared L0/L1/L2 scene composition. Pages own content; this owns the world behind it. */
export function ObservatoryScene({ kind, children, className = "" }: { kind: SceneKind; children: ReactNode; className?: string }) {
  const { resolved } = useTheme();
  const { isMobile } = useBreakpoint();
  const sampled = useSampledSceneColors();
  const appearance = resolved === "dark" ? "night" : "day";
  const style = {
    ...(sampled.day ? { "--scene-sky-seam-day": sampled.day } : {}),
    ...(sampled.night ? { "--scene-sky-seam-night": sampled.night } : {}),
    ...(sampled.aspect ? { "--scene-band-aspect": String(sampled.aspect) } : {}),
  } as CSSProperties;
  return <div className={`observatory scene-${kind} appearance-${appearance} ${isMobile ? "scene-mobile" : "scene-desktop"} ${className}`} style={style}>
    <div className="observatory-sky scene-sky" aria-hidden="true">
      <span className="observatory-sky-layer observatory-sky-day" />
      <span className="observatory-sky-layer observatory-sky-night" />
      <span className="observatory-sky-stars" />
      <span className="scene-constellations" />
      <span className="scene-star-trails" />
      <span className="scene-comets" />
      <span className="scene-balloons" />
      {kind === "plan" && <><span className="scene-lake scene-lake-day" /><span className="scene-lake scene-lake-night" /></>}
    </div>
    {(kind === "plan" || kind === "focus" || kind === "settings") && <div className="scene-plate" aria-hidden="true" />}
    <div className="scene-content">{children}</div>
  </div>;
}
