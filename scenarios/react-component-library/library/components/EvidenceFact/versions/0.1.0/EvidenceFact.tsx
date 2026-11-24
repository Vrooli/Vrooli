/**
 * @libraryId react-component-library:EvidenceFact
 * @displayName Evidence Fact
 * @description Accessible definition-list value with optional evidence disclosure
 * @version 0.1.0
 * @tags ["evidence","data-display"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { useId, useState, type ReactNode } from "react";
import "./EvidenceFact.css";

export interface EvidenceFactProps {
  label: ReactNode;
  value: ReactNode;
  disclosure?: {
    label: string;
    expandedLabel: string;
    content: ReactNode;
  };
}

/** A definition-list value with an optional accessible evidence disclosure. */
export function EvidenceFact({ label, value, disclosure }: EvidenceFactProps) {
  const [open, setOpen] = useState(false);
  const panelId = useId();
  return (
    <div data-rcl-evidence-fact="">
      <dt data-rcl-evidence-fact-label="">{label}</dt>
      <dd data-rcl-evidence-fact-detail="">
        <span data-rcl-evidence-fact-value="">{value}</span>
        {disclosure ? (
          <button
            type="button"
            data-rcl-evidence-fact-toggle=""
            aria-expanded={open}
            aria-controls={panelId}
            onClick={() => setOpen((current) => !current)}
          >
            {open ? disclosure.expandedLabel : disclosure.label}
          </button>
        ) : null}
        {disclosure ? (
          <div id={panelId} data-rcl-evidence-fact-content="" hidden={!open}>
            {disclosure.content}
          </div>
        ) : null}
      </dd>
    </div>
  );
}

export default EvidenceFact;
