import { useLayoutEffect, useMemo, useRef, useState } from "react";
import { AmbientDisplayShell } from "../AmbientDisplayShell";
import { RoomVisual } from "../RoomVisual";
import type { BoardRoom, CatalogEntry, Catalogs } from "../../lib/api";
import { sampleReadings, themeStyle } from "../../lib/settingsPreview";

const str = (value: unknown): string => (typeof value === "string" ? value : "");
const PREVIEW_WIDTH = 1440;
const PREVIEW_HEIGHT = 900;

/**
 * The live preview. It runs the real scene engine over authored signal samples,
 * recolored to the working theme and bound by the working room, so what the
 * operator sees here is what the board renders for the same configuration.
 */
export function SettingsPreview({ catalogs, room, theme, beatIndex = 0 }: { catalogs: Catalogs; room: CatalogEntry | undefined; theme: CatalogEntry | undefined; beatIndex?: number }) {
  const stageRef = useRef<HTMLDivElement>(null);
  const [scale, setScale] = useState(1);
  const signals = catalogs.signals;
  const readings = useMemo(() => sampleReadings(signals ?? []), [signals]);
  const beats = Array.isArray(room?.beats) ? room.beats as Array<Record<string, unknown>> : [];
  const composition = str(beats[beatIndex]?.composition) || str(room?.composition) || (catalogs.compositions?.[0]?.id ?? "orbital-field");
  const previewRoom = useMemo(() => ({ ...(room ?? {}), id: room?.id ?? "settings-preview", title: str(room?.title) || "Settings preview", composition, beats, bind: room?.bind ?? {} }) as unknown as BoardRoom, [beats, composition, room]);
  const style = themeStyle(theme);

  useLayoutEffect(() => {
    const stage = stageRef.current;
    if (!stage) return undefined;

    const updateScale = () => {
      const { width, height } = stage.getBoundingClientRect();
      if (width > 0 && height > 0) {
        setScale(Math.min(width / PREVIEW_WIDTH, height / PREVIEW_HEIGHT));
      }
    };

    updateScale();
    if (typeof ResizeObserver === "undefined") return undefined;
    const observer = new ResizeObserver(updateScale);
    observer.observe(stage);
    return () => observer.disconnect();
  }, []);

  return (
    <div ref={stageRef} className="cc-settings-stage" style={style} data-theme={str(theme?.id) || undefined} data-testid="settings-preview-stage">
      <div
        className="cc-settings-stage-viewport"
        data-fixed-viewport="desktop"
        data-testid="settings-preview-viewport"
        style={{ width: PREVIEW_WIDTH, height: PREVIEW_HEIGHT, transform: `scale(${scale})` }}
      >
        <AmbientDisplayShell
          theme={str(theme?.id) || str(previewRoom.theme) || "ground-control"}
          title={previewRoom.title}
          position="ROOM PREVIEW"
          samples="mark"
          beatIndex={beatIndex}
          beatCount={beats.length}
          beatDurations={beats.map((beat) => typeof beat.dwellSeconds === "number" ? beat.dwellSeconds : 1)}
          manageDocumentTheme={false}
        >
          <RoomVisual room={previewRoom} roomId={previewRoom.id} beatIndex={beatIndex} readings={readings} samples="mark" logicalViewport={{ width: PREVIEW_WIDTH, height: PREVIEW_HEIGHT }} />
        </AmbientDisplayShell>
      </div>
      <p className="cc-settings-stage-caption">Section {beatIndex + 1} · {composition}{str(theme?.id) ? ` · ${str(theme?.id)}` : ""}</p>
    </div>
  );
}
