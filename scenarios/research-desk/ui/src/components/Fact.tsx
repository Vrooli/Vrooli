import type { ReactNode } from "react";
import { UnknownValue } from "@vrooli/react-component-library/UnknownValue/0.1.0";
import { EvidenceFact as RclEvidenceFact } from "@vrooli/react-component-library/EvidenceFact/0.1.0";

import type { Evidence } from "@vrooli/proto-types/research-desk/v1/shared/evidence_pb";

import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { absolute, toDate } from "../lib/format";
import { PathText } from "./PathText";

/**
 * One derived value with its source one tap away, so "why does it say this"
 * is always answerable: every value is derived from files or says why not. A value that could not be derived
 * reads `unknown` with the reason, never a guess.
 */
export function Fact({ label, evidence, children }: { label: ReactNode; evidence?: Evidence | Evidence[]; children: ReactNode }) {
  const { t } = useTranslation();
  const list = (Array.isArray(evidence) ? evidence : evidence ? [evidence] : []).filter(Boolean);
  const known = list.length === 0 || list.some((e) => e.known);
  const disclosure = list.length > 0 ? {
    label: t(strings.evidence.why),
    expandedLabel: t(strings.evidence.hide),
    content: (
      <ul className="rd-evidence">
        {list.map((e, n) => (
          <EvidenceRow key={n} evidence={e} />
        ))}
      </ul>
    ),
  } : undefined;
  return (
    <RclEvidenceFact label={<span className="text-app-muted-foreground">{label}</span>} value={known ? children : <Unknown />} disclosure={disclosure} />
  );
}

function EvidenceRow({ evidence }: { evidence: Evidence }) {
  const { t } = useTranslation();
  const asOf = toDate(evidence.asOf);
  return (
    <li>
      {evidence.reason ? <span className="block">{evidence.reason}</span> : null}
      {evidence.source ? (
        <span className="block">
          {t(strings.evidence.source)} {evidence.source.startsWith("/") ? <PathText path={evidence.source} /> : <span className="rd-path">{evidence.source}</span>}
        </span>
      ) : null}
      {asOf ? <span className="block text-app-muted-foreground">{t(strings.evidence.asOf, { time: absolute(asOf) })}</span> : null}
    </li>
  );
}

export function Unknown({ reason }: { reason?: string }) {
  const { t } = useTranslation();
  return <UnknownValue label={t(strings.common.unknown)} reason={reason} />;
}
