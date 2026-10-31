import { Suspense, useState, type ReactNode } from "react";
import { Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";

/** Download closed surfaces on first use; retain their state and exit animations. */
export default function DeferredSurface({ active, children }: { active: boolean; children: ReactNode }) {
  const { t } = useTranslation();
  const [opened, setOpened] = useState(active);
  if (active && !opened) setOpened(true);
  if (!active && !opened) return null;
  return <Suspense fallback={active ? (
    <div role="status" className="fixed inset-0 z-wc-drawer grid place-items-center bg-wc-surface-base/60">
      <div className="flex items-center gap-2 rounded-lg bg-wc-surface-raised p-4 text-wc-text-primary">
        <Loader2 className="h-5 w-5 animate-spin" aria-hidden="true" />
        {t(strings.app.loading)}
      </div>
    </div>
  ) : null}>{children}</Suspense>;
}
