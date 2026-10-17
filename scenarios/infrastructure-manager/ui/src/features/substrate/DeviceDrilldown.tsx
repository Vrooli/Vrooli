import { LegendPlate } from "../../components/instrument/LegendPlate";
import { Lamp } from "../../components/instrument/Lamp";
import { RUNG_ORDER, rungToken, signalToken, type Rung } from "../../theme/instrument";
import { worstBlindDays, type DeviceClassNode } from "./model";

/**
 * The per-class drill-down.
 *
 * A board that only shows which parts of the machine are unwatched leaves the
 * reader with a problem and no next step. This panel answers the follow-up for
 * one device class, rung by rung: which authored cell the rung answers, what
 * question that cell asks, what the instrument observed, WHY it is not covered,
 * and what would close it.
 *
 * The remediation line is the reason this exists rather than a tooltip.
 * "Storage anticipation is unmeasurable" is a finding; "commission the smartctl
 * host tool and grant raw block-device access at setup time, because the
 * scenario never escalates at runtime" is an action, and the difference is
 * whether the board changes anything.
 *
 * It also surfaces GRADING state, which is easy to miss and load-bearing: a
 * rung can be measured and still ungraded because no setpoint bar resolves. A
 * reading with no bar is not a passing reading, and the panel says which.
 */

export interface DeviceDrilldownProps {
  node: DeviceClassNode;
  labels: DeviceDrilldownLabels;
}

export interface DeviceDrilldownLabels {
  heading: string;
  ladder: string;
  devices: string;
  blindDevices: string;
  reasonHeading: string;
  remediationHeading: string;
  mechanismHeading: string;
  noRemediation: string;
  ungraded: string;
  provisional: string;
  blockedBy: (rung: string) => string;
  blindFor: (days: number) => string;
  notRead: string;
}

export function DeviceDrilldown({ node, labels }: DeviceDrilldownProps) {
  const blindDays = worstBlindDays(node);

  return (
    <section aria-label={node.deviceClass} className="flex flex-col gap-space-sm">
      <LegendPlate
        as="h3"
        legend={labels.heading}
        aside={<span className="font-mono">{node.deviceClass}</span>}
      />

      <dl className="grid gap-x-space-md gap-y-space-2xs [grid-template-columns:auto_1fr] m-0">
        <DetailRow
          label={labels.devices}
          /* `null` means the source did not read, and renders an em dash. A
             zero here would claim the class was enumerated and found empty. */
          value={node.deviceCount === null ? null : String(node.deviceCount)}
          notRead={labels.notRead}
        />
        <DetailRow
          label={labels.blindDevices}
          value={node.blindDevices === null ? null : String(node.blindDevices)}
          notRead={labels.notRead}
        />
      </dl>

      {blindDays !== null ? <p className="blind-note">{labels.blindFor(blindDays)}</p> : null}

      <LegendPlate as="h4" tag="R1–R5" legend={labels.ladder} />
      <ol className="flex flex-col gap-px bg-app-border border border-app-border rounded-panel overflow-hidden list-none p-0 m-0">
        {RUNG_ORDER.map((rung) => (
          <RungDetailRow key={rung} rung={rung} node={node} labels={labels} />
        ))}
      </ol>
    </section>
  );
}

function RungDetailRow({
  rung,
  node,
  labels,
}: {
  rung: Rung;
  node: DeviceClassNode;
  labels: DeviceDrilldownLabels;
}) {
  const token = rungToken(rung);
  const detail = node.rungs[rung];
  const needsAction = detail.state !== "COVERED" && detail.state !== "NOT_APPLICABLE";

  return (
    <li className="bg-app-surface p-space-sm grid gap-space-2xs [grid-template-columns:auto_1fr] items-start">
      <Lamp
        state={detail.state}
        subject={`${node.deviceClass}, ${token.label}`}
        reason={detail.reason ?? undefined}
        blindDays={detail.blindDays ?? undefined}
      />
      <div className="flex flex-col gap-space-3xs">
        <p className="text-label">
          <span className="font-mono text-signal-covered">{token.tag}</span> {token.label}
          <span className="text-app-subtle-foreground"> — {signalToken(detail.state).label}</span>
          {detail.cellRef ? (
            <span className="font-mono text-body-sm text-app-subtle-foreground"> · {detail.cellRef}</span>
          ) : null}
        </p>
        <p className="text-body-sm text-app-muted-foreground">{detail.question ?? token.question}</p>

        {detail.reason ? (
          <p className="text-body-sm">
            <span className="font-mono text-app-subtle-foreground">{labels.reasonHeading}</span>{" "}
            {detail.reason}
          </p>
        ) : null}

        {detail.mechanism ? (
          <p className="text-body-sm">
            <span className="font-mono text-app-subtle-foreground">{labels.mechanismHeading}</span>{" "}
            <span className="font-mono">{detail.mechanism}</span>
          </p>
        ) : null}

        {/* A rung suppressed by a blind foundation says which rung suppressed
            it, so the reader fixes the cause rather than the symptom. */}
        {detail.blockedBy ? (
          <p className="text-body-sm text-app-muted-foreground">
            {labels.blockedBy(rungToken(detail.blockedBy).label)}
          </p>
        ) : null}

        {needsAction ? (
          <p className="text-body-sm">
            <span className="font-mono text-app-subtle-foreground">
              {labels.remediationHeading}
            </span>{" "}
            {detail.remediation ?? (
              <span className="text-app-subtle-foreground">{labels.noRemediation}</span>
            )}
          </p>
        ) : null}

        {/* Ungraded is not passing. A measured reading with no bar behind it has
            not been judged, and saying so is the difference between a reading
            and a verdict. */}
        {!detail.graded && detail.state !== "NOT_APPLICABLE" ? (
          <p className="blind-note">
            {labels.ungraded}
            {detail.ungradedReason ? ` — ${detail.ungradedReason}` : ""}
          </p>
        ) : null}

        {detail.provisional ? <p className="blind-note">{labels.provisional}</p> : null}
      </div>
    </li>
  );
}

function DetailRow({
  label,
  value,
  notRead,
}: {
  label: string;
  value: string | null;
  notRead: string;
}) {
  return (
    <>
      <dt className="font-mono text-body-sm text-app-subtle-foreground uppercase tracking-[0.08em]">
        {label}
      </dt>
      <dd className="m-0 font-mono tabular-nums">
        {value ?? <span className="text-app-subtle-foreground" title={notRead}>—</span>}
      </dd>
    </>
  );
}
