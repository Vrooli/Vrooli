/**
 * ConceptExplainerDialog
 *
 * Generic "explain a concept" surface used wherever the UI offers an info
 * affordance next to a label or section. Each section is a label + body pair
 * with an optional swatch (for visual glossaries like the phase-graph
 * legend). Reuses the shared <Dialog> primitive — no new modal chrome.
 *
 * Canonical concept-explainer in this codebase: do not introduce a parallel
 * implementation. Future "explain X" surfaces should configure this dialog.
 */

import type { ReactNode } from "react";
import { Dialog } from "./dialog";
import { selectors } from "../../consts/selectors";

export interface ConceptExplainerSection {
  /** Short label rendered as a column or term. */
  label: string;
  /** Prose body — string or rich React content. */
  body: ReactNode;
  /** Optional visual swatch (e.g., color chip) shown next to the label. */
  swatch?: ReactNode;
  /** Optional sub-heading rendered above this section. */
  heading?: string;
}

export interface ConceptExplainerDialogProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  /** Optional intro paragraph shown above the section list. */
  intro?: ReactNode;
  /** Optional canonical documentation link for the concept. */
  docLink?: { href: string; label: string };
  sections: ConceptExplainerSection[];
  /** Override the default testId — used by thin wrappers that preserve a
   *  legacy testid (e.g., phase-graph-glossary-dialog). */
  testId?: string;
  /** Override the default Tailwind max-width. */
  maxWidth?: string;
}

export function ConceptExplainerDialog({
  isOpen,
  onClose,
  title,
  intro,
  docLink,
  sections,
  testId = selectors.goalDetails.conceptExplainerDialog,
  maxWidth = "max-w-2xl",
}: ConceptExplainerDialogProps) {
  // Group sections by heading. Sections without a heading land in the
  // implicit "default" group rendered first.
  const groups = new Map<string, ConceptExplainerSection[]>();
  for (const section of sections) {
    const key = section.heading ?? "";
    const list = groups.get(key);
    if (list) {
      list.push(section);
    } else {
      groups.set(key, [section]);
    }
  }

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      title={title}
      maxWidth={maxWidth}
      testId={testId}
    >
      <div className="space-y-6 text-sm text-slate-300">
        {intro ? <p className="leading-relaxed text-slate-300">{intro}</p> : null}
        {docLink ? (
          <a
            href={docLink.href}
            className="inline-flex text-sm font-medium text-cyan-300 hover:text-cyan-200 hover:underline"
          >
            {docLink.label}
          </a>
        ) : null}
        {Array.from(groups.entries()).map(([heading, items]) => (
          <section key={heading || "_default"}>
            {heading ? (
              <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">
                {heading}
              </h3>
            ) : null}
            <ul className="space-y-3">
              {items.map((section) => (
                <li key={`${heading}-${section.label}`} className="grid grid-cols-[7rem_1fr] items-start gap-3">
                  <span className="flex items-center gap-2 text-sm font-medium text-slate-100">
                    {section.swatch}
                    <span>{section.label}</span>
                  </span>
                  <div className="leading-relaxed text-slate-300">{section.body}</div>
                </li>
              ))}
            </ul>
          </section>
        ))}
      </div>
    </Dialog>
  );
}
