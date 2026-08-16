import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { FormSection } from "../components/FormSection";
import { DirtyStateGuard } from "../components/DirtyStateGuard";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { DataTable } from "../components/ui/data-table";
import { useSurfaceState } from "../hooks/useSurfaceState";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { configuredBookId, fetchBooks, fetchPostings, fetchStatement } from "../api/ledger";
import { formatCurrency } from "../i18n/format";

const today = () => new Date().toISOString().slice(0, 10);
const thirtyDaysAgo = () => new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10);

export function StatementsPage() {
  const { t } = useTranslation();
  const books = useQuery({ queryKey: ["books"], queryFn: fetchBooks, retry: false });
  const bookId = configuredBookId() || books.data?.books[0]?.id || "";
  const [period, setPeriod] = useState({ from: thirtyDaysAgo(), to: today() });
  const [appliedPeriod, setAppliedPeriod] = useState(period);
  const statement = useQuery({ queryKey: ["statement", bookId, appliedPeriod.from, appliedPeriod.to], queryFn: () => fetchStatement(bookId, appliedPeriod.from, appliedPeriod.to), retry: false, enabled: Boolean(bookId) });
  const postings = useQuery({ queryKey: ["statement-postings", bookId], queryFn: () => fetchPostings(bookId), retry: false, enabled: Boolean(bookId) });
  const hasPeriodPostings = Boolean(postings.data?.postings.some((posting) => {
    const occurredAt = posting.event?.occurredAt;
    if (!occurredAt) return true;
    const occurred = new Date(Number(occurredAt.seconds) * 1000 + Math.floor(occurredAt.nanos / 1_000_000)).toISOString();
    return (!appliedPeriod.from || occurred >= `${appliedPeriod.from}T00:00:00.000Z`) && (!appliedPeriod.to || occurred <= `${appliedPeriod.to}T23:59:59.999Z`);
  }));
  const emptyPeriod = postings.isSuccess && !hasPeriodPostings && !statement.data?.partial;
  const [exportMessage, setExportMessage] = useState("");
  const statementAvailability = Array.isArray(statement.data?.availability) ? statement.data.availability : [];
  const surface = useSurfaceState({
    query: { isLoading: books.isLoading || statement.isLoading || postings.isLoading, isFetching: books.isFetching || statement.isFetching || postings.isFetching, isError: books.isError || statement.isError || postings.isError, error: books.error || statement.error || postings.error },
    empty: emptyPeriod || (statement.data === undefined && !statement.isLoading && !statement.isError),
  });
  const value = (amount: bigint) => statement.data?.partial ? t(strings.pages.accounts.balanceUnavailable) : formatCurrency(Number(amount) / 100, statement.data?.currency || books.data?.books[0]?.currency || "USD");
  const statementRows: Array<{ label: string; amount: bigint }> = statement.data ? [
    { label: t(strings.pages.statements.openingCash), amount: statement.data.openingCashMinor },
    { label: t(strings.pages.statements.inflow), amount: statement.data.inflowMinor },
    { label: t(strings.pages.statements.outflow), amount: statement.data.outflowMinor },
    { label: t(strings.pages.statements.closingCash), amount: statement.data.closingCashMinor },
    { label: t(strings.pages.statements.revenue), amount: statement.data.revenueMinor },
    { label: t(strings.pages.statements.expense), amount: statement.data.expenseMinor },
    { label: t(strings.pages.statements.assets), amount: statement.data.assetsMinor },
    { label: t(strings.pages.statements.liabilities), amount: statement.data.liabilitiesMinor },
  ] : [];
  const statementColumns = [
    { id: "category", header: t(strings.pages.statements.categoryBreakdown), accessor: (row: typeof statementRows[number]) => row.label, searchValue: (row: typeof statementRows[number]) => row.label, className: "break-words" },
    { id: "amount", header: statement.data?.currency || "USD", accessor: (row: typeof statementRows[number]) => value(row.amount), className: "break-words tabular-nums" },
    { id: "coverage", header: t(strings.pages.statements.coverageNote), accessor: () => t(strings.pages.statements.coverageNote), className: "break-words" },
  ];
  const exportStatement = () => {
    setExportMessage("");
    if (!statement.data) {
      setExportMessage(t(strings.pages.statements.exportError));
      return;
    }
    const payload = JSON.stringify({ ...statement.data, openingCashMinor: statement.data.openingCashMinor.toString(), inflowMinor: statement.data.inflowMinor.toString(), outflowMinor: statement.data.outflowMinor.toString(), closingCashMinor: statement.data.closingCashMinor.toString(), revenueMinor: statement.data.revenueMinor.toString(), expenseMinor: statement.data.expenseMinor.toString(), assetsMinor: statement.data.assetsMinor.toString(), liabilitiesMinor: statement.data.liabilitiesMinor.toString() }, null, 2);
    const url = URL.createObjectURL(new Blob([payload], { type: "application/json" }));
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = `money-ledger-statement-${appliedPeriod.from}-${appliedPeriod.to}.json`;
    anchor.click();
    URL.revokeObjectURL(url);
    setExportMessage(t(strings.pages.statements.exported));
  };
  const applyPeriod = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setAppliedPeriod(period);
  };

  return (
    <ExperienceSurface surfaceId="statements" state={surface.state} statusMessage={surface.reason} data-testid={selectors.pages.statements} aria-labelledby="statements-heading" className="flex flex-col gap-4">
      <h2 id="statements-heading" className="text-2xl font-semibold">{t(strings.pages.statements.title)}</h2>
      <Card data-testid={selectors.pages.statementFigures}>
        <CardHeader><CardTitle>{t(strings.pages.statements.cardTitle)}</CardTitle></CardHeader>
        <CardContent>
          <p className="text-app-muted-foreground">{t(strings.pages.statements.description)}</p>
          <DirtyStateGuard isDirty={period.from !== appliedPeriod.from || period.to !== appliedPeriod.to} protectUnload title={t(strings.pages.statements.periodSelector)} description={t(strings.pages.statements.coverageNote)}>
            <div data-testid={selectors.pages.periodSelector} className="mt-4 rounded-md border p-3"><FormSection title={t(strings.pages.statements.periodSelector)}><form className="grid gap-3 sm:grid-cols-3" onSubmit={applyPeriod}>
              <label className="grid gap-1" htmlFor="statement-from"><span>{t(strings.pages.statements.fromLabel)}</span><Input id="statement-from" type="date" value={period.from} onChange={(event) => setPeriod({ ...period, from: event.target.value })} /></label>
              <label className="grid gap-1" htmlFor="statement-to"><span>{t(strings.pages.statements.toLabel)}</span><Input id="statement-to" type="date" value={period.to} onChange={(event) => setPeriod({ ...period, to: event.target.value })} /></label>
              <Button type="submit" className="self-end" disabled={!period.from || !period.to}>{t(strings.pages.statements.applyPeriod)}</Button>
            </form></FormSection></div>
          </DirtyStateGuard>
          <p data-testid={selectors.pages.coverageNote} role="status" className="text-sm text-app-muted-foreground">{t(strings.pages.statements.coverageNote)}</p>
          {statement.data && !emptyPeriod && <>
            <p role="note" className="text-sm text-app-muted-foreground">{statement.data.partial ? t(strings.pages.statements.coverageNote) : `${statement.data.from || appliedPeriod.from} → ${statement.data.to || appliedPeriod.to}`}</p>
            {statementAvailability.length > 0 && <h3 className="mt-3 font-medium">{t(strings.pages.statements.availability)}</h3>}
            {statementAvailability.map((item) => <p key={item.adapterId} role="note" className="text-sm text-amber-700">{item.adapterId}: {item.reason}</p>)}
            <DataTable rows={statementRows} columns={statementColumns} getRowKey={(row) => row.label} caption={t(strings.pages.statements.cardTitle)} searchLabel={t(strings.pages.statements.categoryBreakdown)} searchPlaceholder={t(strings.pages.statements.categoryBreakdown)} emptyMessage={t(strings.pages.statements.emptyGuidance)} tableTestId={selectors.pages.categoryBreakdown} className="mt-4" />
          </>}
          <p data-testid={selectors.pages.uncategorisedCount} role="status" className="text-sm text-app-muted-foreground">{t(strings.pages.statements.uncategorisedCount)}: {t(strings.pages.statements.uncategorisedNotReported)}</p>
          <p data-testid={selectors.pages.notTaxAdvice} role="note" className="text-sm text-app-muted-foreground">{t(strings.pages.statements.notTaxAdvice)}</p>
          <Button type="button" data-testid={selectors.pages.exportAction} className="mt-3" onClick={exportStatement}>{t(strings.pages.statements.exportAction)}</Button>
          {exportMessage && <p role="status" className="text-sm text-app-success">{exportMessage}</p>}
          <p data-testid={selectors.pages.statementsEmptyGuidance} role="note" className={surface.state === "empty" ? "mt-3 rounded-md border border-dashed p-3 text-app-muted-foreground" : "sr-only"}>{t(strings.pages.statements.emptyGuidance)}</p>
        </CardContent>
      </Card>
    </ExperienceSurface>
  );
}
