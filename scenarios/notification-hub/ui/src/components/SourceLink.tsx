import { strings } from "../consts/strings";
import { externalHost } from "../features/asks/askView";
import { useTranslation } from "../i18n";

/**
 * The producer's own page for a notification. A tap reaches it only after the
 * Hub's record, so the destination is named beside the words: the operator sees
 * where it goes before they leave. The host is isolated as left-to-right text so
 * it cannot reorder the label in a right-to-left language.
 */
export function SourceLink({ href, className }: { href: string; className?: string }) {
  const { t } = useTranslation();
  const host = externalHost(href);
  return (
    <a href={href} rel="noreferrer" className={className}>
      {t(strings.pages.dashboard.openSource)}
      {host && <> · <bdi dir="ltr">{host}</bdi></>}
    </a>
  );
}
