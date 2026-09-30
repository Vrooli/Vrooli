import { AlertTriangle, Camera, CheckCircle2 } from "lucide-react";
import { useMemo, useState } from "react";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { Badge } from "./ui/Badge";
import { Card, CardBody, CardDescription, CardHeader, CardTitle } from "./ui/Card";
import { EmptyState } from "./ui/EmptyState";
import { Select } from "./ui/Select";
import { StatusPill } from "./ui/StatusPill";

type CaptureTone = "error" | "warn" | "ok";

interface CaptureViolation {
  code: string;
  message: string;
  severity: CaptureTone;
  edge?: "top" | "right" | "bottom" | "left";
  box: {
    left: number;
    top: number;
    width: number;
    height: number;
  };
}

interface CaptureRecord {
  id: string;
  scenario: string;
  page: string;
  viewport: string;
  imageLabel: string;
  declaredStatusColor: string;
  declaredStatusLabel: string;
  capturedStatusColor: string;
  capturedStatusLabel: string;
  declaredSafeAreaColor: string;
  declaredSafeAreaLabel: string;
  capturedSafeAreaColor: string;
  capturedSafeAreaLabel: string;
  violations: CaptureViolation[];
}

const CAPTURES: CaptureRecord[] = [
  {
    id: "experience-manager-fleet-mobile",
    scenario: "experience-manager",
    page: "/fleet",
    viewport: "390 x 844",
    imageLabel: "Fleet page mobile capture",
    declaredStatusColor: "var(--color-shell)",
    declaredStatusLabel: "shell",
    capturedStatusColor: "var(--color-background)",
    capturedStatusLabel: "background",
    declaredSafeAreaColor: "var(--color-shell)",
    declaredSafeAreaLabel: "shell",
    capturedSafeAreaColor: "var(--color-surface)",
    capturedSafeAreaLabel: "surface",
    violations: [
      {
        code: "visual_status_bar_color_mismatch",
        message: "Status strip differs from declared chrome color.",
        severity: "error",
        edge: "top",
        box: { left: 0, top: 0, width: 100, height: 8 },
      },
      {
        code: "visual_unsafe_edge_tap_zone",
        message: "Primary action overlaps the unsafe top edge.",
        severity: "error",
        edge: "top",
        box: { left: 62, top: 5, width: 28, height: 10 },
      },
    ],
  },
  {
    id: "ui-health-dashboard-desktop",
    scenario: "ui-health",
    page: "/",
    viewport: "1440 x 900",
    imageLabel: "Dashboard desktop capture",
    declaredStatusColor: "var(--color-background)",
    declaredStatusLabel: "background",
    capturedStatusColor: "var(--color-background)",
    capturedStatusLabel: "background",
    declaredSafeAreaColor: "var(--color-background)",
    declaredSafeAreaLabel: "background",
    capturedSafeAreaColor: "var(--color-background)",
    capturedSafeAreaLabel: "background",
    violations: [],
  },
];

const TONE_TO_BADGE: Record<CaptureTone, "error" | "warn" | "success"> = {
  error: "error",
  warn: "warn",
  ok: "success",
};

export function CaptureGallery() {
  const { t } = useTranslation();
  const scenarioOptions = useMemo(() => {
    const scenarios = Array.from(new Set(CAPTURES.map((capture) => capture.scenario))).sort();
    return [
      { value: "all", label: t(strings.pages.captures.filters.allScenarios) },
      ...scenarios.map((scenario) => ({ value: scenario, label: scenario })),
    ];
  }, [t]);
  const [scenario, setScenario] = useState("all");
  const captures = CAPTURES.filter((capture) => scenario === "all" || capture.scenario === scenario);

  return (
    <>
      <div className="flex flex-col gap-2 sm:max-w-xs">
        <label htmlFor="captures-scenario" className="text-sm font-medium">
          {t(strings.pages.captures.filters.scenario)}
        </label>
        <Select
          id="captures-scenario"
          ariaLabel={t(strings.pages.captures.filters.scenario)}
          value={scenario}
          onChange={setScenario}
          options={scenarioOptions}
          data-testid={selectors.captures.scenarioSelect}
        />
      </div>

      {captures.length === 0 ? (
        <EmptyState
          icon={Camera}
          title={t(strings.pages.captures.empty)}
          data-testid={selectors.captures.empty}
        />
      ) : (
        <div data-testid={selectors.captures.gallery} className="grid gap-4 xl:grid-cols-2">
          {captures.map((capture) => (
            <CaptureCard key={capture.id} capture={capture} />
          ))}
        </div>
      )}
    </>
  );
}

function CaptureCard({ capture }: { capture: CaptureRecord }) {
  const { t } = useTranslation();
  const status = capture.violations.length > 0 ? "error" : "ok";
  return (
    <Card data-testid={selectors.captures.captureCard({ captureId: capture.id })}>
      <CardHeader className="flex flex-row items-start justify-between gap-3">
        <div>
          <CardTitle>{capture.scenario}</CardTitle>
          <CardDescription>
            {capture.page} · {capture.viewport}
          </CardDescription>
        </div>
        <StatusPill
          status={status}
          label={
            status === "ok"
              ? t(strings.pages.captures.status.clean)
              : t(strings.pages.captures.status.findings, { count: capture.violations.length })
          }
          icon={status === "ok" ? CheckCircle2 : AlertTriangle}
        />
      </CardHeader>
      <CardBody>
        <div className="grid gap-4 lg:grid-cols-[minmax(15rem,22rem)_1fr]">
          <DeviceFrame capture={capture} />
          <div className="flex flex-col gap-4">
            <ChromeComparison capture={capture} />
            <ViolationList violations={capture.violations} />
          </div>
        </div>
      </CardBody>
    </Card>
  );
}

function DeviceFrame({ capture }: { capture: CaptureRecord }) {
  return (
    <figure
      className="relative mx-auto aspect-[9/16] w-full max-w-[22rem] overflow-hidden rounded-sheet border border-app-border bg-app-shell p-2 shadow-sm"
      aria-label={capture.imageLabel}
    >
      <div className="h-full overflow-hidden rounded-panel bg-app-background">
        <div
          className="h-[7%]"
          style={{ backgroundColor: capture.capturedStatusColor }}
          aria-hidden
        />
        <div
          className="relative h-[93%] overflow-hidden"
          style={{ backgroundColor: capture.capturedSafeAreaColor }}
        >
          <div className="m-3 flex h-[calc(100%-1.5rem)] flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-3">
            <div className="h-5 w-1/2 rounded-control bg-app-surface-muted" />
            <div className="grid grid-cols-3 gap-2">
              <div className="h-14 rounded-control bg-app-primary/15" />
              <div className="h-14 rounded-control bg-app-info/15" />
              <div className="h-14 rounded-control bg-app-warning/15" />
            </div>
            <div className="flex flex-1 flex-col gap-2">
              <div className="h-7 rounded-control bg-app-surface-muted" />
              <div className="h-7 rounded-control bg-app-surface-muted" />
              <div className="h-7 rounded-control bg-app-surface-muted" />
              <div className="h-7 rounded-control bg-app-surface-muted" />
            </div>
          </div>
        </div>
      </div>
      {capture.violations.map((violation, index) => (
        <ViolationOverlay key={`${violation.code}-${index}`} violation={violation} index={index} />
      ))}
    </figure>
  );
}

function ViolationOverlay({ violation, index }: { violation: CaptureViolation; index: number }) {
  const toneClass =
    violation.severity === "warn"
      ? "border-app-warning bg-app-warning/20"
      : "border-app-danger bg-app-danger/20";
  return (
    <div
      data-testid={selectors.captures.overlay({ index })}
      className={`absolute rounded-control border-2 ${toneClass}`}
      style={{
        left: `${violation.box.left}%`,
        top: `${violation.box.top}%`,
        width: `${violation.box.width}%`,
        height: `${violation.box.height}%`,
      }}
      aria-hidden
    />
  );
}

function ChromeComparison({ capture }: { capture: CaptureRecord }) {
  const { t } = useTranslation();
  return (
    <div className="grid gap-3 sm:grid-cols-2">
      <ColorPair
        label={t(strings.pages.captures.chrome.statusBar)}
        declaredColor={capture.declaredStatusColor}
        declaredLabel={capture.declaredStatusLabel}
        capturedColor={capture.capturedStatusColor}
        capturedLabel={capture.capturedStatusLabel}
      />
      <ColorPair
        label={t(strings.pages.captures.chrome.safeArea)}
        declaredColor={capture.declaredSafeAreaColor}
        declaredLabel={capture.declaredSafeAreaLabel}
        capturedColor={capture.capturedSafeAreaColor}
        capturedLabel={capture.capturedSafeAreaLabel}
      />
    </div>
  );
}

function ColorPair({
  label,
  declaredColor,
  declaredLabel,
  capturedColor,
  capturedLabel,
}: {
  label: string;
  declaredColor: string;
  declaredLabel: string;
  capturedColor: string;
  capturedLabel: string;
}) {
  const { t } = useTranslation();
  return (
    <div className="rounded-panel border border-app-border bg-app-surface-muted p-3">
      <p className="text-xs font-medium uppercase tracking-wide text-app-muted-foreground">{label}</p>
      <dl className="mt-3 grid grid-cols-2 gap-2 text-xs">
        <ColorSwatch label={t(strings.pages.captures.chrome.declared)} color={declaredColor} value={declaredLabel} />
        <ColorSwatch label={t(strings.pages.captures.chrome.captured)} color={capturedColor} value={capturedLabel} />
      </dl>
    </div>
  );
}

function ColorSwatch({ label, color, value }: { label: string; color: string; value: string }) {
  return (
    <div className="flex flex-col gap-1">
      <dt className="text-app-muted-foreground">{label}</dt>
      <dd className="flex items-center gap-2">
        <span
          className="h-5 w-5 rounded-control border border-app-border"
          style={{ backgroundColor: color }}
          aria-hidden
        />
        <span className="font-mono text-[0.7rem] text-app-foreground">{value}</span>
      </dd>
    </div>
  );
}

function ViolationList({ violations }: { violations: CaptureViolation[] }) {
  const { t } = useTranslation();
  if (violations.length === 0) {
    return (
      <EmptyState
        icon={CheckCircle2}
        title={t(strings.pages.captures.violations.empty)}
        className="p-6"
      />
    );
  }
  return (
    <ul data-testid={selectors.captures.overlayList} className="flex flex-col gap-2">
      {violations.map((violation, index) => (
        <li
          key={`${violation.code}-${index}`}
          className="rounded-panel border border-app-border bg-app-surface p-3"
        >
          <div className="flex flex-wrap items-center gap-2">
            <Badge tone={TONE_TO_BADGE[violation.severity]}>{violation.severity}</Badge>
            <span className="font-mono text-xs text-app-muted-foreground">{violation.code}</span>
            {violation.edge ? (
              <Badge tone="neutral">{t(strings.pages.captures.violations.edge, { edge: violation.edge })}</Badge>
            ) : null}
          </div>
          <p className="pt-2 text-sm text-app-foreground">{violation.message}</p>
        </li>
      ))}
    </ul>
  );
}
