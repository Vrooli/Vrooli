import { ExternalLink } from "lucide-react";

import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/**
 * A research-home HTML file (a review with embedded media) opens in its own
 * tab. The desk's file route serves it sandboxed (scripts off, opaque origin)
 * and refuses framing, so it is never shown inside the desk.
 */
export function HtmlOpen({ src, name }: { src: string; name: string }) {
  const { t } = useTranslation();
  return (
    <a href={src} target="_blank" rel="noreferrer noopener" className="rd-open">
      <ExternalLink aria-hidden="true" className="size-5 shrink-0" />
      <span className="flex min-w-0 flex-col">
        <span className="rd-id">{name}</span>
        <span className="text-body-sm text-app-muted-foreground">{t(strings.files.htmlOpens)}</span>
      </span>
    </a>
  );
}
