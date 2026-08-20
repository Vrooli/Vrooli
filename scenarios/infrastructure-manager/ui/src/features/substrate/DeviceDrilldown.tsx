import { LegendPlate } from "../../components/instrument/LegendPlate";
import { Lamp } from "../../components/instrument/Lamp";
import { RUNG_ORDER, rungToken, signalToken, type Rung } from "../../theme/instrument";
import { type DeviceNode, worstBlindDays } from "./model";

/**
 * The per-device drill-down.
 *
 * A board that only shows which devices are unwatched leaves the reader with a
 * problem and no next step. This panel answers the follow-up questions for one
 * device: what is it, what is known about it, where its evidence goes, what
 * controls it, and — for every rung that is not covered — WHY, and what would
 * close it.
 *
 * The remediation line is the reason this panel exists rather than a tooltip.
 * "SMART is unmeasurable" is a finding; "commission the smartctl host tool and
 * grant raw block-device access at setup time, because the scenario never
 * escalates at runtime" is an action, and the difference is whether the board
 * changes anything.
 */

export interface DeviceDrilldownProps {
  device: DeviceNode;
  /** Section tag, e.g. the device class's cell reference. */
  tag?: string;
  labels: DeviceDrilldownLabels;
}

export interface DeviceDrilldownLabels {
  heading: string;
  identity: string;
  ladder: string;
  provenance: string;
  vendor: string;
  driver: string;
  parent: string;
  nodes: string;
  discoveredBy: string;
  enrichedBy: string;
  reasonHeading: string;
  remediationHeading: string;
  noRemediation: string;
  blindFor: (days: number) => string;
}

export function DeviceDrilldown({ device, tag, labels }: DeviceDrilldownProps) {
  const blindDays = worstBlindDays(device);

  return (
    <section aria-label={device.name} className="flex flex-col gap-space-sm">
      <LegendPlate
        as="h3"
        tag={tag}
        legend={labels.heading}
        aside={<span className="font-mono">{device.id}</span>}
      />

      <dl className="grid gap-x-space-md gap-y-space-2xs [grid-template-columns:auto_1fr] m-0">
        <DetailRow label={labels.identity} value={device.name} />
        <DetailRow label={labels.vendor} value={device.vendor} />
        <DetailRow label={labels.driver} value={device.driver} />
        <DetailRow label={labels.parent} value={device.parent} mono />
        <DetailRow label={labels.nodes} value={device.nodes.join(", ") || null} mono />
        <DetailRow label={labels.discoveredBy} value={device.discoveredBy} mono />
        <DetailRow label={labels.enrichedBy} value={device.enrichedBy.join(", ") || null} mono />
      </dl>

      {blindDays !== null ? <p className="blind-note">{labels.blindFor(blindDays)}</p> : null}

      <LegendPlate as="h4" tag="R1–R5" legend={labels.ladder} />
      <ol className="flex flex-col gap-px bg-app-border border border-app-border rounded-panel overflow-hidden list-none p-0 m-0">
        {RUNG_ORDER.map((rung) => (
          <RungDetail key={rung} rung={rung} device={device} labels={labels} />
        ))}
      </ol>
    </section>
  );
}

function RungDetail({
  rung,
  device,
  labels,
}: {
  rung: Rung;
  device: DeviceNode;
  labels: DeviceDrilldownLabels;
}) {
  const token = rungToken(rung);
  const state = device.rungs[rung];
  const reason = device.reasons[rung];
  const remediation = device.remediation[rung];
  const needsAction = state !== "COVERED";

  return (
    <li className="bg-app-surface p-space-sm grid gap-space-2xs [grid-template-columns:auto_1fr] items-start">
      <Lamp
        state={state}
        subject={`${device.name}, ${token.label}`}
        reason={reason}
        blindDays={device.blindDays[rung]}
      />
      <div className="flex flex-col gap-space-3xs">
        <p className="text-label">
          <span className="font-mono text-signal-covered">{token.tag}</span> {token.label}
          <span className="text-app-subtle-foreground"> — {signalToken(state).label}</span>
        </p>
        <p className="text-body-sm text-app-muted-foreground">{token.question}</p>
        {reason ? (
          <p className="text-body-sm">
            <span className="font-mono text-app-subtle-foreground">
              {labels.reasonHeading}
            </span>{" "}
            {reason}
          </p>
        ) : null}
        {needsAction ? (
          <p className="text-body-sm">
            <span className="font-mono text-app-subtle-foreground">
              {labels.remediationHeading}
            </span>{" "}
            {remediation ?? (
              <span className="text-app-subtle-foreground">{labels.noRemediation}</span>
            )}
          </p>
        ) : null}
      </div>
    </li>
  );
}

function DetailRow({
  label,
  value,
  mono = false,
}: {
  label: string;
  value: string | null;
  mono?: boolean;
}) {
  return (
    <>
      <dt className="font-mono text-body-sm text-app-subtle-foreground uppercase tracking-[0.08em]">
        {label}
      </dt>
      {/*
        An absent field renders an em dash, never an empty cell and never a
        plausible default. "This device declares no driver" and "we did not
        record a driver" are different facts, and the collector's provenance
        fields above are where a reader distinguishes them.
      */}
      <dd className={`m-0 ${mono ? "font-mono text-body-sm" : ""}`}>
        {value ?? <span className="text-app-subtle-foreground">—</span>}
      </dd>
    </>
  );
}
