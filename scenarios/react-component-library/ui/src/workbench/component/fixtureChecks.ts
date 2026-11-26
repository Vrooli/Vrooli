import {
  buildContrastRow,
  buildTouchRow,
  buildTypeNote,
  type ContextCheckRow,
  type Translate,
} from "./contextChecks";
import { componentProbeContrast, type ComponentProbe } from "./componentProbe";
import type { FixturePaint } from "./fixtureBench";

/** The selected frame's measured size, unique to the fixture bench. */
function buildSizeRow(paint: FixturePaint | null, t: Translate): ContextCheckRow {
  const measured = Boolean(paint && paint.width > 0 && paint.height > 0);
  return {
    id: "size",
    label: t("workbench.component.fixtureCheckSize", { defaultValue: "Size fits its frame" }),
    detail: measured
      ? t("workbench.component.fixtureCheckSizeDetail", {
          defaultValue: "{{width}} × {{height}} px measured; the frame is drawn at this size.",
          width: paint?.width ?? 0,
          height: paint?.height ?? 0,
        })
      : t("workbench.component.fixtureCheckMeasuring", { defaultValue: "Measuring this frame…" }),
    verdict: measured ? "pass" : "unmeasured",
    scored: true,
    group: "fit",
  };
}

/**
 * The compact component-fit rows for one fixture. The selected frame has no
 * page, so this is deliberately the same fit scope as In Context without the
 * page, evidence or human rows.
 */
export function buildFixtureChecks(
  paint: FixturePaint | null,
  probe: ComponentProbe | null,
  t: Translate,
): ContextCheckRow[] {
  const measured = Boolean(probe?.measured);
  const targets = measured ? probe?.targets : null;
  const type = measured ? (probe?.typography ?? null) : null;
  const contrast = componentProbeContrast(probe);

  return [
    buildSizeRow(paint, t),
    buildTouchRow(t, {
      on: measured,
      total: targets?.total ?? 0,
      small: targets?.small ?? 0,
      pending: !measured,
    }),
    buildContrastRow(t, {
      on: contrast.on,
      ratio: contrast.ratio,
      label: contrast.label,
      background: contrast.background,
      noText: measured && !type,
      pending: !measured,
    }),
    buildTypeNote(t, {
      measured,
      typography: type,
      pending: !measured,
    }),
  ];
}
