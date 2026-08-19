import { useState, type FormEvent, type KeyboardEvent } from "react";
import { CheckCircle2, Clock3, ShieldCheck, XCircle } from "lucide-react";

import {
  ApprovalStatus,
  listPendingApprovals,
  resolveApproval,
  type ApprovalRequest,
} from "../api/approvals";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { StatusBadge } from "../components/ui/status-badge";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

export function DashboardPage() {
  const { t, i18n } = useTranslation();
  const [operatorToken, setOperatorToken] = useState("");
  const [approvals, setApprovals] = useState<ApprovalRequest[]>([]);
  const [loading, setLoading] = useState(false);
  const [authenticated, setAuthenticated] = useState(false);
  const [error, setError] = useState("");
  const [announcement, setAnnouncement] = useState("");
  const [resolvingId, setResolvingId] = useState("");
  const [bookId, setBookId] = useState("");

  async function openQueue(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      const pending = await listPendingApprovals(operatorToken);
      setApprovals(pending);
      setBookId([...new Set(pending.map((approval) => approval.bookId).filter(Boolean))].sort()[0] ?? "");
      setAuthenticated(true);
    } catch {
      setAuthenticated(false);
      setError(t(strings.approvals.authError));
    } finally {
      setLoading(false);
    }
  }

  const books = [...new Set(approvals.map((approval) => approval.bookId).filter(Boolean))].sort();
  const visibleApprovals = bookId
    ? approvals.filter((approval) => approval.bookId === bookId)
    : approvals;

  async function decide(approval: ApprovalRequest, resolution: ApprovalStatus.APPROVED | ApprovalStatus.DECLINED) {
    setResolvingId(approval.id);
    setError("");
    try {
      await resolveApproval(operatorToken, approval.id, resolution);
      setApprovals((current) => current.filter((item) => item.id !== approval.id));
      setAnnouncement(t(
        resolution === ApprovalStatus.APPROVED
          ? strings.approvals.approvedAnnouncement
          : strings.approvals.declinedAnnouncement,
        { counterparty: approval.counterparty },
      ));
    } catch {
      setError(t(strings.approvals.resolveError));
    } finally {
      setResolvingId("");
    }
  }

  function decideFromKeyboard(
    event: KeyboardEvent<HTMLButtonElement>,
    approval: ApprovalRequest,
    resolution: ApprovalStatus.APPROVED | ApprovalStatus.DECLINED,
  ) {
    if (event.key !== "Enter" && event.key !== " ") return;
    event.preventDefault();
    void decide(approval, resolution);
  }

  return (
    <section
      data-testid={selectors.pages.approvals}
      aria-labelledby="approvals-heading"
      className="mx-auto flex w-full max-w-6xl flex-col gap-6"
    >
      <header className="flex items-start gap-3">
        <ShieldCheck aria-hidden="true" className="mt-1 size-7 shrink-0 text-app-primary" />
        <div>
          <h2 id="approvals-heading" className="text-2xl font-semibold">
            {t(strings.approvals.title)}
          </h2>
          <p className="mt-1 max-w-3xl text-app-muted-foreground">
            {t(strings.approvals.description)}
          </p>
        </div>
      </header>

      {!authenticated ? (
        <Card className="max-w-xl">
          <CardHeader><CardTitle>{t(strings.approvals.signInTitle)}</CardTitle></CardHeader>
          <CardContent>
            <form className="flex flex-col gap-4" onSubmit={(event) => void openQueue(event)}>
              <div className="space-y-2">
                <label htmlFor="operator-token" className="text-sm font-medium">
                  {t(strings.approvals.tokenLabel)}
                </label>
                <Input
                  id="operator-token"
                  data-testid={selectors.approvals.tokenInput}
                  type="password"
                  autoComplete="current-password"
                  required
                  value={operatorToken}
                  onChange={(event) => setOperatorToken(event.target.value)}
                />
                <p className="text-sm text-app-muted-foreground">{t(strings.approvals.tokenHelp)}</p>
              </div>
              <Button data-testid={selectors.approvals.openQueueButton} type="submit" disabled={loading}>
                {loading ? t(strings.approvals.loading) : t(strings.approvals.openQueue)}
              </Button>
            </form>
          </CardContent>
        </Card>
      ) : (
        <div data-testid={selectors.approvals.queue} className="space-y-4">
          {books.length > 0 ? (
            <div className="max-w-md space-y-2">
              <label htmlFor="approval-book" className="text-sm font-medium">
                {t(strings.approvals.bookLabel)}
              </label>
              <select
                id="approval-book"
                data-testid={selectors.approvals.bookSelect}
                className="min-h-11 w-full rounded-md border border-app-border bg-app-card px-3 text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-ring"
                value={bookId}
                onChange={(event) => setBookId(event.target.value)}
              >
                {books.map((book) => <option key={book} value={book}>{book}</option>)}
              </select>
            </div>
          ) : null}
          <div className="flex items-center justify-between gap-3">
            <h3 className="text-lg font-semibold">{t(strings.approvals.pendingTitle)}</h3>
            <StatusBadge tone="warning">
              <Clock3 aria-hidden="true" className="me-1 size-3.5" />
              {t(strings.approvals.pendingCount, { count: visibleApprovals.length })}
            </StatusBadge>
          </div>
          {visibleApprovals.length === 0 ? (
            <Card><CardContent className="py-8 text-center text-app-muted-foreground">{t(strings.approvals.empty)}</CardContent></Card>
          ) : (
            <ul className="grid gap-4" aria-label={t(strings.approvals.pendingTitle)}>
              {visibleApprovals.map((approval) => {
                const money = formatMoney(approval.amountMinor, approval.currency, i18n.language);
                const busy = resolvingId === approval.id;
                return (
                  <li key={approval.id} data-testid={selectors.approvals.item({ id: approval.id })}>
                    <Card className="border-s-4 border-s-app-warning">
                      <CardHeader className="gap-3 sm:flex-row sm:items-start sm:justify-between">
                        <div>
                          <CardTitle className="text-xl tabular-nums">{money}</CardTitle>
                          <p className="mt-1 font-medium">{approval.counterparty}</p>
                        </div>
                        <StatusBadge tone="warning">
                          <Clock3 aria-hidden="true" className="me-1 size-3.5" />
                          {t(strings.approvals.statusQueued)}
                        </StatusBadge>
                      </CardHeader>
                      <CardContent className="grid gap-4">
                        <dl className="grid gap-3 text-sm sm:grid-cols-3">
                          <ApprovalDetail label={t(strings.approvals.agentLabel)} value={approval.requestingAgent} />
                          <ApprovalDetail label={t(strings.approvals.bookLabel)} value={approval.bookId} />
                          <ApprovalDetail label={t(strings.approvals.mandateLabel)} value={approval.mandateId} />
                          <ApprovalDetail label={t(strings.approvals.expiresLabel)} value={formatTimestamp(approval.expiresAt, i18n.language)} />
                        </dl>
                        <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
                          <Button
                            data-testid={selectors.approvals.action({ id: approval.id, action: "decline" })}
                            variant="danger"
                            disabled={busy}
                            aria-label={t(strings.approvals.declineAria, { amount: money, counterparty: approval.counterparty })}
                            onKeyDown={(event) => decideFromKeyboard(event, approval, ApprovalStatus.DECLINED)}
                            onClick={() => void decide(approval, ApprovalStatus.DECLINED)}
                          >
                            <XCircle aria-hidden="true" className="size-4" />
                            {t(strings.approvals.decline)}
                          </Button>
                          <Button
                            data-testid={selectors.approvals.action({ id: approval.id, action: "approve" })}
                            disabled={busy}
                            aria-label={t(strings.approvals.approveAria, { amount: money, counterparty: approval.counterparty })}
                            onKeyDown={(event) => decideFromKeyboard(event, approval, ApprovalStatus.APPROVED)}
                            onClick={() => void decide(approval, ApprovalStatus.APPROVED)}
                          >
                            <CheckCircle2 aria-hidden="true" className="size-4" />
                            {t(strings.approvals.approve)}
                          </Button>
                        </div>
                      </CardContent>
                    </Card>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      )}

      {error && <p role="alert" className="rounded-control border border-app-danger/40 bg-app-danger/10 p-3 text-sm font-medium text-app-danger">{error}</p>}
      <p role="status" aria-live="polite" className="sr-only">{announcement}</p>
    </section>
  );
}

function ApprovalDetail({ label, value }: { label: string; value: string }) {
  return <div><dt className="text-app-muted-foreground">{label}</dt><dd className="mt-1 break-all font-medium">{value || "—"}</dd></div>;
}

function formatMoney(amountMinor: bigint, currency: string, locale: string): string {
  return new Intl.NumberFormat(locale, { style: "currency", currency: currency || "USD" }).format(Number(amountMinor) / 100);
}

function formatTimestamp(value: { seconds: bigint } | undefined, locale: string): string {
  if (!value) return "—";
  return new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" }).format(Number(value.seconds) * 1000);
}
