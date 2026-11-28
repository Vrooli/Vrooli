import { X } from "lucide-react";

import { useTranslation } from "../../i18n";
import { useFocusTrap } from "../useFocusTrap";
import { REVIEW_HELP_INTRO, REVIEW_HELP_SECTIONS, REVIEW_HELP_TITLE } from "./reviewHelp";

/**
 * Review's help surface. The page itself stays a working queue; the contract
 * and the decision semantics live here, one click away, instead of occupying
 * the first screen every time someone opens Review.
 */
export function ReviewHelpDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useTranslation();
  const layer = useFocusTrap<HTMLDivElement>(open, onClose);

  if (!open) return null;

  return (
    <div
      ref={layer}
      className="wb-help-layer"
      role="dialog"
      aria-modal="true"
      aria-labelledby="review-help-title"
    >
      <div className="wb-help-panel">
        <div className="wb-help-head">
          <h2 id="review-help-title">{REVIEW_HELP_TITLE}</h2>
          <button
            type="button"
            className="wb-iconbtn"
            onClick={onClose}
            aria-label={t("workbench.review.helpClose", { defaultValue: "Close Review help" })}
          >
            <X aria-hidden size={15} />
          </button>
        </div>
        <div className="wb-help-body">
          <p className="wb-help-intro">{REVIEW_HELP_INTRO}</p>
          {REVIEW_HELP_SECTIONS.map((section) => (
            <section key={section.title} className="wb-help-section">
              <h3>{section.title}</h3>
              {section.body ? <p>{section.body}</p> : null}
              {section.bullets ? (
                <ul>
                  {section.bullets.map((bullet) => (
                    <li key={bullet}>{bullet}</li>
                  ))}
                </ul>
              ) : null}
            </section>
          ))}
        </div>
      </div>
    </div>
  );
}
