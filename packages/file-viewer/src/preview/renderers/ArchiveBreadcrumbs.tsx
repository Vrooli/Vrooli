import { useEffect, useRef } from "react";
import { ChevronRight } from "lucide-react";

import { useFileViewerText } from "../../i18n/context.js";
import { strings } from "../../i18n/strings.js";

export function ArchiveBreadcrumbs({ folder, onNavigate }: { folder: string; onNavigate?: (folder: string) => void }) {
  const { t } = useFileViewerText(), c = strings.archivePreview;
  const parts = folder.split("/").filter(Boolean);
  const navigation = useRef<HTMLDivElement>(null);
  useEffect(() => { navigation.current?.querySelector("[aria-current=location]")?.scrollIntoView({ block: "nearest", inline: "end" }); }, [folder]);
  const buttonClass = "min-h-11 shrink-0 rounded px-2 text-fv-text-secondary outline-none hover:text-fv-text-primary focus-visible:ring-2 focus-visible:ring-fv-accent";
  return (
    <div ref={navigation} role="group" aria-label={t(c.location)} data-testid="archive-breadcrumbs" className="flex min-h-11 min-w-0 flex-1 items-center overflow-x-auto text-xs">
      <button type="button" aria-current={parts.length === 0 ? "location" : undefined} onClick={() => { onNavigate?.(""); }} className={buttonClass}>{t(c.archive)}</button>
      {parts.map((part, index) => {
        const path = parts.slice(0, index + 1).join("/");
        return <span key={path} className="flex shrink-0 items-center">
          <ChevronRight aria-hidden="true" className="h-3 w-3 text-fv-text-faint rtl:rotate-180" />
          <button type="button" aria-current={index === parts.length - 1 ? "location" : undefined}
            onClick={() => { onNavigate?.(path); }} className={`${buttonClass} max-w-52 truncate`}><bdi>{part}</bdi></button>
        </span>;
      })}
    </div>
  );
}
