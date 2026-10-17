import { strings } from "../consts/strings.generated";
import { useTranslation } from "../i18n";
import { LegendPlate } from "../components/instrument/LegendPlate";
import { Lamp, LampLegend } from "../components/instrument/Lamp";
import { RungRing, describeRungStates } from "../components/instrument/RungRing";
import { StatPlate, StatStrip } from "../components/instrument/StatPlate";
import { AnnunciatorGrid, type AnnunciatorRow } from "../components/instrument/AnnunciatorGrid";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { RUNG_ORDER, rungToken, signalToken, type Rung, type SignalState } from "../theme/instrument";

/**
 * The instrument-language fixture.
 *
 * Every primitive, in every state, on one strings.pages.designLanguage. It is a route rather than a
 * test artifact for two reasons: the language can be reviewed in one view
 * before a real surface consumes it, and a visual regression in a primitive
 * becomes visible without reading a diff.
 *
 * The data on this page is deliberately FIXED. Nothing here reads the host.
 * A style fixture that reads live data stops being a fixture the first time
 * the host changes, and its caption says so plainly so no reader mistakes
 * these rows for a measurement.
 */

const ALL_STATES: readonly SignalState[] = [
  "COVERED",
  "PARTIAL",
  "EXCURSION",
  "UNMEASURABLE",
  "UNAVAILABLE",
  "NOT_APPLICABLE",
  "UNAUTHORED",
  "BLIND",
  "SOURCE_DOWN",
];

const ring = (...states: SignalState[]): Record<Rung, SignalState> =>
  RUNG_ORDER.reduce(
    (acc, rung, index) => {
      acc[rung] = states[index] ?? "BLIND";
      return acc;
    },
    {} as Record<Rung, SignalState>,
  );

const RING_FIXTURES: readonly { label: string; states: Record<Rung, SignalState> }[] = [
  { label: "Fully covered", states: ring("COVERED", "COVERED", "COVERED", "COVERED", "COVERED") },
  { label: "Blind at the top", states: ring("COVERED", "COVERED", "COVERED", "COVERED", "BLIND") },
  { label: "Telemetry only", states: ring("COVERED", "COVERED", "BLIND", "BLIND", "BLIND") },
  { label: "Unmeasurable tip", states: ring("COVERED", "COVERED", "PARTIAL", "COVERED", "UNMEASURABLE") },
  { label: "Not enumerated", states: ring("BLIND", "BLIND", "BLIND", "BLIND", "BLIND") },
  { label: "Mechanism absent", states: ring("COVERED", "UNAVAILABLE", "UNAVAILABLE", "UNAVAILABLE", "UNAVAILABLE") },
  { label: "Graded elsewhere", states: ring("COVERED", "NOT_APPLICABLE", "NOT_APPLICABLE", "NOT_APPLICABLE", "NOT_APPLICABLE") },
  { label: "Source unreachable", states: ring("SOURCE_DOWN", "SOURCE_DOWN", "SOURCE_DOWN", "SOURCE_DOWN", "SOURCE_DOWN") },
];

const GRID_FIXTURES: readonly AnnunciatorRow[] = [
  {
    id: "fixture-nvme",
    name: "NVMe block device",
    tag: "pci:0000:02:00.0",
    states: ring("COVERED", "COVERED", "PARTIAL", "COVERED", "UNMEASURABLE"),
    reasons: { ANTICIPATION: "smartctl present, permission denied" },
  },
  {
    id: "fixture-igpu",
    name: "Integrated graphics",
    tag: "pci:0000:79:00.0",
    states: ring("BLIND", "BLIND", "BLIND", "BLIND", "BLIND"),
    blindDays: { IDENTITY: 114, TELEMETRY: 114, EVIDENCE: 114, CONTROL: 114, ANTICIPATION: 114 },
  },
  {
    id: "fixture-memory",
    name: "Memory controller",
    tag: "edac:mc0",
    states: ring("COVERED", "PARTIAL", "COVERED", "BLIND", "UNMEASURABLE"),
    reasons: { ANTICIPATION: "no EDAC controller registered on this host" },
    blindDays: { CONTROL: 41 },
  },
  {
    id: "fixture-thermal",
    name: "Thermal sensor bank",
    tag: "hwmon:k10temp",
    states: ring("COVERED", "COVERED", "BLIND", "PARTIAL", "EXCURSION"),
    reasons: { ANTICIPATION: "sustained above the authored deadband" },
    blindDays: { EVIDENCE: 7 },
  },
  {
    id: "fixture-offline",
    name: "Device graph source",
    tag: "system-monitor",
    states: ring("SOURCE_DOWN", "SOURCE_DOWN", "SOURCE_DOWN", "SOURCE_DOWN", "SOURCE_DOWN"),
    reasons: RUNG_ORDER.reduce(
      (acc, rung) => {
        acc[rung] = "source did not answer within its 10s deadline";
        return acc;
      },
      {} as Record<Rung, string>,
    ),
  },
];

const BLIND_FIXTURES = [
  { labelKey: strings.pages.designLanguage.blindNotEnumerated, days: 114, opened: "2026-04-28" },
  { labelKey: strings.pages.designLanguage.blindNoPredictive, days: 41, opened: "2026-07-10" },
  { labelKey: strings.pages.designLanguage.blindNoEvidence, days: 7, opened: "2026-08-13" },
] as const;

export function InstrumentLanguagePage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid="page-design-language"
      aria-labelledby="design-language-heading"
      className="flex flex-col gap-space-xl"
    >
      <header className="flex flex-col gap-space-2xs">
        <p className="font-mono text-body-sm uppercase tracking-[0.22em] text-app-subtle-foreground">
          {t(strings.pages.designLanguage.kitId)}
        </p>
        <h1 id="design-language-heading" className="font-display text-display uppercase tracking-[0.06em]">
          {t(strings.pages.designLanguage.title)}
        </h1>
        <p className="max-w-[66ch] text-app-muted-foreground">{t(strings.pages.designLanguage.description)}</p>
      </header>

      {/* ---------------------------------------------------------- plates -- */}
      <ExperienceSurface surfaceId="language-plates" data-testid="language-plates" state="static" aria-labelledby="fixture-plates" className="flex flex-col gap-space-sm">
        <LegendPlate id="fixture-plates" tag="P1" legend={t(strings.pages.designLanguage.plateHeading)} aside="3 variants" />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">{t(strings.pages.designLanguage.plateNote)}</p>
        <div className="flex flex-col gap-space-md">
          <LegendPlate as="h3" tag="SB1" legend="With a real cell tag" aside="7 cells" />
          <LegendPlate as="h3" legend="No reference to carry" />
          <LegendPlate as="h3" tag="R5" legend="Anticipation" aside="0 of 8 devices" />
        </div>
      </ExperienceSurface>

      {/* ----------------------------------------------------------- lamps -- */}
      <ExperienceSurface surfaceId="language-lamps" data-testid="language-lamps" state="static" aria-labelledby="fixture-lamps" className="flex flex-col gap-space-sm">
        <LegendPlate id="fixture-lamps" tag="P2" legend={t(strings.pages.designLanguage.lampHeading)}
          aside={`${ALL_STATES.length} states`} />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">{t(strings.pages.designLanguage.lampNote)}</p>
        <ul className="grid gap-px bg-app-border border border-app-border rounded-panel overflow-hidden list-none p-0 m-0 [grid-template-columns:repeat(auto-fit,minmax(240px,1fr))]">
          {ALL_STATES.map((state) => {
            const token = signalToken(state);
            return (
              <li key={state} className="bg-app-surface p-space-sm flex items-center gap-space-sm">
                <Lamp
                  state={state}
                  subject={token.label}
                  reason={
                    state === "UNMEASURABLE" || state === "SOURCE_DOWN"
                      ? "a stated reason belongs here"
                      : undefined
                  }
                  blindDays={state === "BLIND" ? 114 : undefined}
                />
                <div className="flex flex-col">
                  <span className="text-label">{token.label}</span>
                  <span className="font-mono text-body-sm text-app-subtle-foreground">{token.tone}</span>
                </div>
              </li>
            );
          })}
        </ul>
        <LampLegend states={ALL_STATES} />
      </ExperienceSurface>

      {/* ----------------------------------------------------------- rings -- */}
      <ExperienceSurface surfaceId="language-rings" data-testid="language-rings" state="static" aria-labelledby="fixture-rings" className="flex flex-col gap-space-sm">
        <LegendPlate id="fixture-rings" tag="P3" legend={t(strings.pages.designLanguage.ringHeading)} aside={`${RUNG_ORDER.length} rungs`} />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">{t(strings.pages.designLanguage.ringNote)}</p>
        <div className="panel p-space-md">
          <ul className="grid gap-space-md list-none p-0 m-0 [grid-template-columns:repeat(auto-fit,minmax(150px,1fr))]">
            {RING_FIXTURES.map((fixture, index) => (
              <li key={fixture.label} className="flex flex-col items-center gap-space-2xs">
                <svg viewBox="0 0 100 100" className="w-full max-w-[120px] h-auto" role="img" aria-label={describeRungStates(fixture.label, fixture.states)}>
                  <RungRing cx={50} cy={50} radius={36} states={fixture.states} animationDelay={index * 0.08} />
                  <circle cx={50} cy={50} r={25} fill="var(--color-surface-raised)" stroke="var(--color-border-lit)" />
                </svg>
                <span className="text-center text-body-sm text-app-muted-foreground">{fixture.label}</span>
              </li>
            ))}
          </ul>
          <ol className="mt-space-md flex flex-wrap gap-x-space-md gap-y-space-3xs list-none p-0 pt-space-sm border-t border-app-border">
            {RUNG_ORDER.map((rung) => (
              <li key={rung} className="font-mono text-body-sm text-app-muted-foreground">
                <span className="text-signal-covered">{rungToken(rung).tag}</span> {rungToken(rung).label}
              </li>
            ))}
          </ol>
        </div>
      </ExperienceSurface>

      {/* ----------------------------------------------------------- stats -- */}
      <ExperienceSurface surfaceId="language-stats" data-testid="language-stats" state="static" aria-labelledby="fixture-stats" className="flex flex-col gap-space-sm">
        <LegendPlate id="fixture-stats" tag="P4" legend={t(strings.pages.designLanguage.statHeading)} aside="3 tones" />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">{t(strings.pages.designLanguage.statNote)}</p>
        <StatStrip label={t(strings.pages.designLanguage.statHeading)}>
          <StatPlate value="27.5%" label={t(strings.pages.designLanguage.statRungsCovered)} tone="covered" />
          <StatPlate value="22" label={t(strings.pages.designLanguage.statCellsBlind)} tone="excursion" />
          <StatPlate value="8" label={t(strings.pages.designLanguage.statDevices)} />
          <StatPlate value={null} label={t(strings.pages.designLanguage.statNotComputable)} />
        </StatStrip>
      </ExperienceSurface>

      {/* ------------------------------------------------------------ grid -- */}
      <ExperienceSurface surfaceId="language-grid" data-testid="language-grid" state="static" aria-labelledby="fixture-grid" className="flex flex-col gap-space-sm">
        <LegendPlate id="fixture-grid" tag="P5" legend={t(strings.pages.designLanguage.gridHeading)} aside={`${GRID_FIXTURES.length} rows`} />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">{t(strings.pages.designLanguage.gridNote)}</p>
        <AnnunciatorGrid rows={GRID_FIXTURES} rowHeader="Device" caption={t(strings.pages.designLanguage.fixtureCaption)} />
      </ExperienceSurface>

      {/* -------------------------------------------------------- blindness -- */}
      <ExperienceSurface surfaceId="language-blindness" data-testid="language-blindness" state="static" aria-labelledby="fixture-blind" className="flex flex-col gap-space-sm">
        <LegendPlate id="fixture-blind" tag="P6" legend={t(strings.pages.designLanguage.blindHeading)} />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">{t(strings.pages.designLanguage.blindNote)}</p>
        <div className="panel p-space-md flex flex-col gap-space-2xs">
          {BLIND_FIXTURES.map((fixture) => (
            <p className="blind-note" key={fixture.labelKey}>
              {t(fixture.labelKey)}
              {" · "}
              <span className="blind-note__age">{t(strings.pages.designLanguage.blindAgeDays, { days: fixture.days })}</span>
              {" · "}
              {t(strings.pages.designLanguage.blindOpened, { date: fixture.opened })}
            </p>
          ))}
        </div>
      </ExperienceSurface>
    </section>
  );
}
