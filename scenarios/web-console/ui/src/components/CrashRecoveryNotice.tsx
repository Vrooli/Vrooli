import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import { listRecoverableSessions } from "../api/sessions";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";

interface CrashRecoveryNoticeProps {
  onOpenArchive: () => void;
  topSafe?: boolean;
}

export default function CrashRecoveryNotice({ onOpenArchive, topSafe = false }: CrashRecoveryNoticeProps) {
  const { t } = useTranslation();
  const [count, setCount] = useState(0);

  const refresh = useCallback(() => {
    void listRecoverableSessions().then((rows) => {
      setCount(rows.length);
    }).catch(() => {
      setCount(0);
    });
  }, []);

  useEffect(() => {
    refresh();
    window.addEventListener("web-console:recoverable-changed", refresh);
    return () => {
      window.removeEventListener("web-console:recoverable-changed", refresh);
    };
  }, [refresh]);

  if (count === 0) return null;
  return <div data-testid="crash-recovery-notice" className={cn("wc-stable-theme flex items-center gap-2 border-b border-amber-700/40 bg-amber-900/20 px-3 py-1.5 text-xs text-amber-100", topSafe && "pt-[var(--wc-safe-top,0px)]")}>
    <span className="min-w-0 flex-1 font-medium">{t(strings.recoverableSessions.heading, { count })}</span>
    <button type="button" onClick={onOpenArchive} className="rounded border border-amber-400/50 px-2 py-1">{t(strings.recoverableSessions.viewArchive)}</button>
  </div>;
}
