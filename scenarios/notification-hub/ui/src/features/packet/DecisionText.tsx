import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { renderDecisionText, type TextLinks } from "./decisionLinks";

interface DecisionTextProps {
  value: string;
  className?: string;
  /** Inherit the surrounding type size (headings) instead of body text. */
  inheritText?: boolean;
  /** Lay links out inline at the line height, for a heading, instead of as 44px touch targets. */
  dense?: boolean;
  links?: TextLinks;
}

/** Decision prose as a phone-safe, wrapping span. It takes its own direction (`dir="auto"`): the sender may write in another language than the page. */
export function DecisionText({ value, className = "", inheritText = false, dense = false, links }: DecisionTextProps) {
  const { t } = useTranslation();
  return (
    <span dir="auto" className={`whitespace-pre-wrap break-words ${inheritText ? "" : "text-sm"} ${className}`}>
      {renderDecisionText(value, {
        ...links,
        dense,
        openAttachmentTitle: t(strings.packet.text.openAttachment),
        openLinkedArtifactTitle: t(strings.packet.text.openLinkedArtifact),
      })}
    </span>
  );
}
