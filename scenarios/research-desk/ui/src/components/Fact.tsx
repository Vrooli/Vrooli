import { useId, useState, type ReactNode } from "react";

import type { Evidence } from "@vrooli/proto-types/research-desk/v1/shared/evidence_pb";

import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { absolute, toDate } from "../lib/format";
import { PathText } from "./PathText";

/**
 * One derived value with its source one tap away, so "why does it say this"
 * is always answerable (charter G3, I6). A value that could not be derived
 * reads `unknown` with the reason, never a guess.
 */
export function Fact({ label, evidence, children }: { label: ReactNode; evidence?: Evidence | Evidence[]; children: ReactNode }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const panelId = useId();
  const list = (Array.isArray(evidence) ? evidence : evidence ? [evidence] : []).filter(Boolean);
  const known = list.length === 0 || list.some((e) => e.known);
  return (
    <div className="rd-fact">
      <dt className="text-app-muted-foreground">{label}</dt>
      <dd className="min-w-0">
        <span className="break-words">{known ? children : <Unknown />}</span>
        {list.length > 0 ? (
          <button type="button" className="rd-why" aria-expanded={open} aria-controls={panelId} onClick={() => setOpen((v) => !v)}>
            {t(open ? strings.evidence.hide : strings.evidence.why)}
          </button>
        ) : null}
        {open ? (
          <ul id={panelId} className="rd-evidence">
            {list.map((e, n) => (
              <EvidenceRow key={n} evidence={e} />
            ))}
          </ul>
        ) : null}
      </dd>
    </div>
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
  return (
    <span className="rd-unknown" title={reason}>
      {t(strings.common.unknown)}
      {reason ? <span className="text-app-muted-foreground">{` · ${reason}`}</span> : null}
    </span>
  );
}
