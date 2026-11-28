import { ArrowUpRight, Copy, ShieldCheck } from "lucide-react";

import {
  DispositionDecision,
  FindingSeverity,
  FindingStatus,
  type Finding,
} from "../../api/review";
import { Button } from "../../../../library/components/Button/versions/2.2.14/Button";
import { Input } from "../../../../library/components/Input/versions/1.3.6/Input";
import { Select } from "../../../../library/components/Select/versions/1.3.24/Select";
import { Textarea } from "../../../../library/components/Textarea/versions/1.1.1/Textarea";
import { useTranslation } from "../../i18n";
import { Chip, Panel, PanelBody, PanelHead, WbButton } from "../primitives";
import { gateHelp, gateLabel, severityHelp } from "./reviewGateHelp";

const SEVERITY_LABEL: Record<FindingSeverity, string> = {
  [FindingSeverity.UNSPECIFIED]: "Unspecified",
  [FindingSeverity.BLOCKER]: "Blocker",
  [FindingSeverity.HIGH]: "High",
  [FindingSeverity.MEDIUM]: "Medium",
  [FindingSeverity.LOW]: "Low",
  [FindingSeverity.INFO]: "Info",
};

type Props = {
  selected: Finding;
  copied: boolean;
  copyEvidence: (finding: Finding) => Promise<void>;
  decision: DispositionDecision;
  setDecision: (value: DispositionDecision) => void;
  owner: string;
  setOwner: (value: string) => void;
  assignmentPending: boolean;
  onAssign: () => void;
  reason: string;
  setReason: (value: string) => void;
  actor: string;
  setActor: (value: string) => void;
  dispositionPending: boolean;
  onRecord: () => void;
  message: string | null;
};

export function ReviewFindingDetail(props: Props) {
  const { t } = useTranslation();
  const { selected } = props;
  const {
    copied,
    copyEvidence,
    decision,
    setDecision,
    owner,
    setOwner,
    assignmentPending,
    onAssign,
    reason,
    setReason,
    actor,
    setActor,
    dispositionPending,
    onRecord,
    message,
  } = props;
  return (
    <Panel
      role="region"
      aria-label={t("workbench.review.detail", { defaultValue: "Finding detail" })}
    >
      <PanelHead
        title={selected.subjectId}
        meta={`${gateLabel(selected.gate)} check · ${SEVERITY_LABEL[selected.severity]}`}
      />
      <PanelBody>
        <p>{selected.message}</p>
        <div className="wb-review-help">
          <p>
            <b>{t("workbench.review.gateHelp", { defaultValue: "What this check does" })}</b>{" "}
            {gateHelp(selected.gate)}
          </p>
          <p>
            <b>
              {t("workbench.review.severityHelp", { defaultValue: "What this severity means" })}
            </b>{" "}
            {severityHelp(severityKey(selected.severity))}
          </p>
        </div>
        {selected.remediation ? (
          <p className="wb-muted">
            <b>{t("workbench.review.remediation", { defaultValue: "Remediation" })}:</b>{" "}
            {selected.remediation}
          </p>
        ) : null}
        <div className="wb-row" style={{ flexWrap: "wrap", gap: 6 }}>
          <Chip tone={selected.severity === FindingSeverity.BLOCKER ? "red" : "amber"}>
            {SEVERITY_LABEL[selected.severity]}
          </Chip>
          <span className="wb-muted wb-small">
            {t("workbench.review.statusFilter", { defaultValue: "Status" })}:{" "}
            <StatusLabel status={selected.status} />
          </span>
          {selected.owner ? (
            <span className="wb-muted wb-small">
              {t("workbench.review.owner", { defaultValue: "Owner" })}: {selected.owner}
            </span>
          ) : null}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => void copyEvidence(selected)}
          >
            <Copy aria-hidden size={12} />
            {copied
              ? t("workbench.review.copied", { defaultValue: "Copied" })
              : t("workbench.review.copyEvidence", { defaultValue: "Copy evidence" })}
          </Button>
          {selected.docsRef ? (
            <a
              className="wb-btn quiet sm"
              href={`/${selected.docsRef}`}
              target="_blank"
              rel="noreferrer"
            >
              {t("workbench.review.docs", { defaultValue: "Docs" })}
              <ArrowUpRight aria-hidden size={12} />
            </a>
          ) : null}
        </div>

        <details className="wb-muted" style={{ marginTop: 10 }}>
          <summary>{t("workbench.review.evidence", { defaultValue: "Evidence" })}</summary>
          <pre style={{ whiteSpace: "pre-wrap", marginTop: 6 }}>
            {prettyJson(selected.evidenceJson)}
          </pre>
        </details>

        <div style={{ marginTop: 14 }}>
          <span className="wb-caption">
            {t("workbench.review.history", { defaultValue: "Decision history" })}
          </span>
          {selected.dispositions.length === 0 ? (
            <p className="wb-muted-2 wb-small">
              {t("workbench.review.none", { defaultValue: "None" })}
            </p>
          ) : (
            selected.dispositions.map((entry) => (
              <div key={entry.id} className="wb-finding">
                <b>
                  <DecisionLabel decision={entry.decision} />
                </b>
                <small>
                  {entry.actor} · {entry.at}
                </small>
                <p>{entry.reason}</p>
              </div>
            ))
          )}
        </div>

        <div className="wb-stack" style={{ marginTop: 14 }}>
          <span className="wb-caption">
            {t("workbench.review.decision", { defaultValue: "Needs a decision" })}
          </span>
          <label className="wb-field">
            <span>{t("workbench.review.owner", { defaultValue: "Owner" })}</span>
            <Input
              value={owner}
              onChange={(event) => setOwner(event.target.value)}
              placeholder={
                selected.owner
                  ? t("workbench.review.ownerCurrent", {
                      defaultValue: "Current owner: {{owner}}",
                      owner: selected.owner,
                    })
                  : t("workbench.review.ownerPlaceholder", {
                      defaultValue: "name@example",
                    })
              }
            />
          </label>
          <WbButton
            variant="ghost"
            size="sm"
            disabled={!owner.trim() || assignmentPending}
            onClick={() => onAssign()}
          >
            {t("workbench.review.assign", { defaultValue: "Assign owner" })}
          </WbButton>
          <label className="wb-field">
            <span>{t("workbench.review.decisionLabel", { defaultValue: "Decision" })}</span>
            <Select
              value={decision}
              onChange={(event) => setDecision(Number(event.target.value))}
              options={[
                {
                  value: String(DispositionDecision.FIX),
                  label: t("workbench.review.fix", { defaultValue: "Fix" }),
                },
                {
                  value: String(DispositionDecision.ACKNOWLEDGE),
                  label: t("workbench.review.acknowledge", { defaultValue: "Acknowledge" }),
                },
                {
                  value: String(DispositionDecision.ACCEPT_RISK),
                  label: t("workbench.review.acceptRisk", { defaultValue: "Accept risk" }),
                },
                {
                  value: String(DispositionDecision.RETIRE),
                  label: t("workbench.review.retire", { defaultValue: "Retire" }),
                },
              ]}
            />
          </label>
          <span className="wb-muted-2 wb-small">{decisionHelp(decision)}</span>
          <label className="wb-field">
            <span>{t("workbench.review.reason", { defaultValue: "Reason" })}</span>
            <Textarea rows={3} value={reason} onChange={(event) => setReason(event.target.value)} />
          </label>
          <label className="wb-field">
            <span>{t("workbench.review.actor", { defaultValue: "Actor" })}</span>
            <Input value={actor} onChange={(event) => setActor(event.target.value)} />
          </label>
          <WbButton
            variant="primary"
            size="sm"
            disabled={!reason.trim() || dispositionPending}
            onClick={() => onRecord()}
          >
            <ShieldCheck aria-hidden size={14} />
            {t("workbench.review.record", { defaultValue: "Record decision" })}
          </WbButton>
          {message ? (
            <p className="wb-muted wb-small" role="status">
              {message}
            </p>
          ) : null}
        </div>
      </PanelBody>
    </Panel>
  );
}

function StatusLabel({ status }: { status: FindingStatus }) {
  const { t } = useTranslation();
  switch (status) {
    case FindingStatus.ACKNOWLEDGED:
      return (
        <span>{t("workbench.review.statusAcknowledged", { defaultValue: "Acknowledged" })}</span>
      );
    case FindingStatus.ACCEPTED:
      return <span>{t("workbench.review.statusAccepted", { defaultValue: "Accepted" })}</span>;
    case FindingStatus.FIXED:
      return <span>{t("workbench.review.statusFixed", { defaultValue: "Fixed" })}</span>;
    case FindingStatus.RETIRED:
      return <span>{t("workbench.review.statusRetired", { defaultValue: "Retired" })}</span>;
    default:
      return <span>{t("workbench.review.statusOpen", { defaultValue: "Open" })}</span>;
  }
}

function severityKey(severity: FindingSeverity): string {
  switch (severity) {
    case FindingSeverity.BLOCKER:
      return "blocker";
    case FindingSeverity.HIGH:
      return "high";
    case FindingSeverity.MEDIUM:
      return "medium";
    case FindingSeverity.LOW:
      return "low";
    case FindingSeverity.INFO:
      return "info";
    default:
      return "medium";
  }
}

function prettyJson(value: string): string {
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}

function decisionHelp(decision: DispositionDecision): string {
  switch (decision) {
    case DispositionDecision.ACKNOWLEDGE:
      return "Records that the team knows about it and owns it. It leaves the open queue and is counted as acknowledged.";
    case DispositionDecision.ACCEPT_RISK:
      return "Records that the team will live with the failure. It leaves the open queue, but the reason is kept as the acceptance record.";
    case DispositionDecision.RETIRE:
      return "Archives the item as no longer relevant. Use this when the asset or version is gone, not to silence a live failure.";
    default:
      return "Records an intent to repair. The item stays open until a later check passes.";
  }
}

function DecisionLabel({ decision }: { decision: DispositionDecision }) {
  const { t } = useTranslation();
  switch (decision) {
    case DispositionDecision.ACKNOWLEDGE:
      return <>{t("workbench.review.acknowledge", { defaultValue: "Acknowledge" })}</>;
    case DispositionDecision.ACCEPT_RISK:
      return <>{t("workbench.review.acceptRisk", { defaultValue: "Accept risk" })}</>;
    case DispositionDecision.FIX:
      return <>{t("workbench.review.fix", { defaultValue: "Fix" })}</>;
    case DispositionDecision.RETIRE:
      return <>{t("workbench.review.retire", { defaultValue: "Retire" })}</>;
    default:
      return <>{t("workbench.review.decisionLabel", { defaultValue: "Decision" })}</>;
  }
}
