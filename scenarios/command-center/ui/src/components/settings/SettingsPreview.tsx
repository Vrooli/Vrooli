import { useMemo } from "react";
import { AmbientCanvas } from "../AmbientCanvas";
import type { CatalogEntry, Catalogs } from "../../lib/api";
import { resolveSlotBindings } from "../../lib/catalogs";
import { sampleReadings, themeStyle } from "../../lib/settingsPreview";

const str = (value: unknown): string => (typeof value === "string" ? value : "");

/**
 * The live preview. It runs the real scene engine over authored signal samples,
 * recolored to the working theme and bound by the working room, so what the
 * operator sees here is what the board renders for the same configuration.
 */
export function SettingsPreview({ catalogs, room, theme }: { catalogs: Catalogs; room: CatalogEntry | undefined; theme: CatalogEntry | undefined }) {
  const signals = catalogs.signals;
  const readings = useMemo(() => sampleReadings(signals ?? []), [signals]);
  const composition = str(room?.composition) || (catalogs.compositions?.[0]?.id ?? "orbital-field");
  const bindKey = JSON.stringify(room?.bind ?? {});
  const slotBindings = useMemo(() => {
    const bind = room?.bind && typeof room.bind === "object" ? (room.bind as Record<string, string>) : {};
    return resolveSlotBindings(composition, (signals ?? []).map((signal) => signal.id), bind);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [composition, signals, bindKey]);
  const style = themeStyle(theme);
  return (
    <div className="cc-settings-stage" style={style} data-theme={str(theme?.id) || undefined} data-testid="settings-preview-stage">
      <div className="cc-settings-stage-ground" aria-hidden="true" />
      <AmbientCanvas composition={composition} readings={readings} quietRefs={[]} forcedTier="reduced" seed="settings-preview" slotBindings={slotBindings} />
      <p className="cc-settings-stage-caption">{composition}{str(theme?.id) ? ` · ${str(theme?.id)}` : ""}</p>
    </div>
  );
}
