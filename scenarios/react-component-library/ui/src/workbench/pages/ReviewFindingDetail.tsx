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
        meta={`${selected.gate} · ${SEVERITY_LABEL[selected.severity]}`}
      />
      <PanelBody>
        <p>{selected.message}</p>
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
          <span className="wb-tag">
            <StatusLabel status={selected.status} />
          </span>
          {selected.owner ? <span className="wb-tag">{selected.owner}</span> : null}
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
          <pre style={{ whiteSpace: "pre-wrap", marginTop: 6 }}>{selected.evidenceJson}</pre>
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
              value={owner || selected.owner}
              onChange={(event) => setOwner(event.target.value)}
              placeholder={t("workbench.review.ownerPlaceholder", {
                defaultValue: "name@example",
              })}
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
                { value: String(DispositionDecision.FIX), label: t("workbench.review.fix", { defaultValue: "Fix" }) },
                { value: String(DispositionDecision.ACKNOWLEDGE), label: t("workbench.review.acknowledge", { defaultValue: "Acknowledge" }) },
                { value: String(DispositionDecision.ACCEPT_RISK), label: t("workbench.review.acceptRisk", { defaultValue: "Accept risk" }) },
                { value: String(DispositionDecision.RETIRE), label: t("workbench.review.retire", { defaultValue: "Retire" }) },
              ]}
            />
          </label>
          <label className="wb-field">
            <span>{t("workbench.review.reason", { defaultValue: "Reason" })}</span>
            <Textarea
              rows={3}
              value={reason}
              onChange={(event) => setReason(event.target.value)}
            />
          </label>
          <label className="wb-field">
            <span>{t("workbench.review.actor", { defaultValue: "Actor" })}</span>
            <Input
              value={actor}
              onChange={(event) => setActor(event.target.value)}
            />
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
      return <>{t("workbench.review.statusAcknowledged", { defaultValue: "Acknowledged" })}</>;
    case FindingStatus.ACCEPTED:
      return <>{t("workbench.review.statusAccepted", { defaultValue: "Accepted" })}</>;
    case FindingStatus.FIXED:
      return <>{t("workbench.review.statusFixed", { defaultValue: "Fixed" })}</>;
    case FindingStatus.RETIRED:
      return <>{t("workbench.review.statusRetired", { defaultValue: "Retired" })}</>;
    default:
      return <>{t("workbench.review.statusOpen", { defaultValue: "Open" })}</>;
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
