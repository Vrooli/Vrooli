import type { ReactNode } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";

import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { isUnauthenticated } from "../lib/apiError";
import { SignedOutCard } from "../features/session/SignedOutCard";

/**
 * The one way a list says it could not load. A refused session shows the
 * sign-in card in its place; anything else names the problem, says nothing was
 * lost, and offers a retry.
 */
export function QueryError({ error, title, detail, onRetry, retrying, retryTestId, children }: {
  error: unknown;
  title: string;
  detail?: string;
  onRetry?: () => void;
  retrying?: boolean;
  retryTestId?: string;
  children?: ReactNode;
}) {
  const { t } = useTranslation();
  if (isUnauthenticated(error)) return <SignedOutCard />;
  return (
    <div role="alert" className="flex flex-col items-start gap-3 rounded-panel border border-app-border border-s-4 border-s-app-danger bg-app-surface p-4">
      <div className="flex flex-col gap-1">
        <p className="text-base font-semibold">{title}</p>
        {detail && <p className="text-sm text-app-muted-foreground">{detail}</p>}
      </div>
      {children}
      {onRetry && <Button type="button" variant="outline" size="sm" data-testid={retryTestId} disabled={retrying} onClick={onRetry}>{t(strings.common.retry)}</Button>}
    </div>
  );
}

/** A quiet one-line note for something that failed without taking the list down. */
export function InlineNote({ children, tone = "muted" }: { children: ReactNode; tone?: "muted" | "warning" }) {
  return <p role="status" className={tone === "warning" ? "text-sm text-app-foreground" : "text-sm text-app-muted-foreground"}>{children}</p>;
}

/** An empty list: an icon, a plain sentence of fact, and at most one next step. */
export function EmptyNotice({ icon, title, detail, children }: { icon: ReactNode; title: string; detail?: string; children?: ReactNode }) {
  return (
    <div className="flex flex-col items-center gap-2 rounded-panel border border-dashed border-app-border px-6 py-10 text-center">
      <span className="text-app-muted-foreground" aria-hidden="true">{icon}</span>
      <h2 className="text-lg font-semibold">{title}</h2>
      {detail && <p className="max-w-sm text-sm text-app-muted-foreground">{detail}</p>}
      {children}
    </div>
  );
}
