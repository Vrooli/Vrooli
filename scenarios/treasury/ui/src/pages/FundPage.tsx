import { useCallback, useEffect, useId, useRef, useState, type FormEvent } from "react";
import { ArrowDownToLine, ArrowUpFromLine, Clipboard, ExternalLink, RefreshCw, ShieldCheck } from "lucide-react";
import { QRCodeSVG } from "qrcode.react";

import { createFundingIntent, confirmFundingTransfer, executeWithdrawal, getLedgerFundingDestination, listFundingIntents, previewWithdrawal, type FundingIntent, type LedgerDestination, type Withdrawal, type WithdrawalQuote } from "../api/funding";
import { getOperatorBudgetHeadroom, getOperatorSession, refreshOperatorSession, signInOperator, signOutOperator, type OperatorSession } from "../api/operatorSession";
import type { AgentBudgetHeadroom } from "@vrooli/proto-types/treasury/v1/authorization/authorization_pb";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1.3.1";
import { Input } from "@vrooli/react-component-library/Input/1.3.6";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

type FundingForm = {
  budgetId: string;
  amount: string;
  value: string;
};

export function FundPage() {
  const { t } = useTranslation();
  const [operatorSession, setOperatorSession] = useState<OperatorSession>({ authenticated: false });
  const [sessionLoading, setSessionLoading] = useState(true);
  const [sessionBusy, setSessionBusy] = useState(false);
  const [loginEmail, setLoginEmail] = useState("");
  const [loginPassword, setLoginPassword] = useState("");
  const [mfaChallenge, setMfaChallenge] = useState("");
  const [mfaCode, setMfaCode] = useState("");
  const [recoveryCode, setRecoveryCode] = useState("");
  const [form, setForm] = useState<FundingForm>({ budgetId: "", amount: "", value: "" });
  const [intents, setIntents] = useState<FundingIntent[]>([]);
  const [activeIntent, setActiveIntent] = useState<FundingIntent>();
  const [headroom, setHeadroom] = useState<AgentBudgetHeadroom>();
  const [headroomError, setHeadroomError] = useState("");
  const [ledgerDestination, setLedgerDestination] = useState<LedgerDestination>();
  const [ledgerDestinationError, setLedgerDestinationError] = useState("");
  const [transactionHash, setTransactionHash] = useState("");
  const [destination, setDestination] = useState("");
  const [mandateId, setMandateId] = useState("");
  const [withdrawalAmount, setWithdrawalAmount] = useState("");
  const [withdrawalValue, setWithdrawalValue] = useState("");
  const [feeWei, setFeeWei] = useState("0");
  const [feeValue, setFeeValue] = useState("0");
  const [quote, setQuote] = useState<WithdrawalQuote>();
  const [withdrawal, setWithdrawal] = useState<Withdrawal>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const withdrawalIdempotencyKeys = useRef(new Map<string, string>());

  useEffect(() => {
    let stopped = false;
    void getOperatorSession().then(async (session) => {
      if (!session.authenticated) {
        try {
          const refreshed = await refreshOperatorSession();
          session = refreshed.authenticated ? await getOperatorSession() : refreshed;
        } catch {
          // A signed-out browser has no refresh cookie; sign-in remains available.
        }
      }
      if (!stopped) setOperatorSession(session);
    }).catch((reason) => {
      if (!stopped) setError(errorText(reason, t(strings.fund.sessionError)));
    }).finally(() => {
      if (!stopped) setSessionLoading(false);
    });
    return () => { stopped = true; };
  }, [t]);

  const refreshHeadroom = useCallback(async () => {
    const budgetId = form.budgetId.trim();
    if (!operatorSession.authenticated || !budgetId) {
      setHeadroom(undefined);
      setHeadroomError("");
      return;
    }
    try {
      setHeadroom(await getOperatorBudgetHeadroom(budgetId));
      setHeadroomError("");
    } catch (reason) {
      setHeadroom(undefined);
      setHeadroomError(errorText(reason, t(strings.fund.headroomError)));
    }
  }, [form.budgetId, operatorSession.authenticated, t]);

  useEffect(() => {
    if (!operatorSession.authenticated) {
      setLedgerDestination(undefined);
      setLedgerDestinationError("");
      return;
    }

    let stopped = false;
    const timer = setTimeout(() => {
      void getLedgerFundingDestination().then((destination) => {
        if (stopped) return;
        setLedgerDestination(destination);
        setLedgerDestinationError("");
      }).catch((reason) => {
        if (stopped) return;
        setLedgerDestination(undefined);
        setLedgerDestinationError(errorText(reason, t(strings.fund.ledgerDestinationError)));
      });
    }, 250);

    return () => {
      stopped = true;
      clearTimeout(timer);
    };
  }, [operatorSession.authenticated, t]);

  async function refreshIntents() {
    if (!operatorSession.authenticated || !form.budgetId.trim()) return;
    setBusy(true);
    setError("");
    try {
      const updated = await listFundingIntents(form.budgetId.trim());
      setIntents(updated);
      setActiveIntent((current) => updated.find((item) => item.id === current?.id) ?? current);
      await refreshHeadroom();
    } catch (reason) {
      setError(errorText(reason, t(strings.fund.requestError)));
    } finally {
      setBusy(false);
    }
  }

  useEffect(() => {
    if (!operatorSession.authenticated || !form.budgetId.trim() || !activeIntent || !isFundingIntentPending(activeIntent.state)) return;

    let stopped = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const poll = async () => {
      try {
        const updated = await listFundingIntents(form.budgetId.trim());
        if (stopped) return;
        setIntents(updated);
        const latest = updated.find((item) => item.id === activeIntent.id);
        if (latest) setActiveIntent(latest);
        await refreshHeadroom();
      } catch {
        // Keep the last known receipt state visible; the next bounded poll can recover.
      }
      if (!stopped) timer = setTimeout(() => void poll(), 1_000);
    };

    timer = setTimeout(() => void poll(), 1_000);
    return () => {
      stopped = true;
      if (timer) clearTimeout(timer);
    };
  }, [activeIntent, form.budgetId, operatorSession.authenticated, refreshHeadroom]);

  useEffect(() => {
    if (!operatorSession.authenticated || !quote || quote.executionMode !== "local-fork" || !withdrawal || withdrawal.state !== "pending_approval") return;
    const idempotencyKey = withdrawalIdempotencyKeys.current.get(quote.id);
    if (!idempotencyKey) return;

    let stopped = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let attempts = 0;
    const retryApprovedExecution = async () => {
      if (stopped || attempts >= 30) return;
      attempts += 1;
      try {
        // Replaying the original local-fork request is idempotent. Once its
        // explicit approval is resolved, Treasury returns the settled result.
        const latest = await executeWithdrawal(quote.id, idempotencyKey);
        if (stopped) return;
        setWithdrawal(latest);
        if (latest.state === "settled") {
          setError("");
          setNotice(t(strings.fund.withdrawNotice, { state: latest.state }));
          return;
        }
        if (latest.state !== "pending_approval") {
          setError("");
          setNotice(t(strings.fund.withdrawNotice, { state: latest.state }));
          return;
        }
      } catch (reason) {
        if (stopped) return;
        setError(errorText(reason, t(strings.fund.requestError)));
      }
      if (!stopped && attempts < 30) timer = setTimeout(() => void retryApprovedExecution(), 1_000);
    };

    timer = setTimeout(() => void retryApprovedExecution(), 1_000);
    return () => {
      stopped = true;
      if (timer) clearTimeout(timer);
    };
  }, [operatorSession.authenticated, quote?.executionMode, quote?.id, t, withdrawal?.id, withdrawal?.state]);

  async function signIn(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (mfaChallenge && !mfaCode.trim() && !recoveryCode.trim()) {
      setError(t(strings.fund.mfaRequired));
      return;
    }
    setSessionBusy(true);
    setError("");
    setNotice("");
    try {
      const result = await signInOperator({
        email: loginEmail.trim(),
        password: loginPassword,
        mfaChallenge,
        totpCode: mfaCode.trim(),
        recoveryCode: recoveryCode.trim(),
      });
      if (result.mfa_required && result.mfa_challenge) {
        setMfaChallenge(result.mfa_challenge);
        setLoginPassword("");
        setMfaCode("");
        setRecoveryCode("");
        setNotice(t(strings.fund.mfaPrompt));
        return;
      }
      if (!result.authenticated) throw new Error(t(strings.fund.sessionError));
      const session = await getOperatorSession();
      if (!session.authenticated) throw new Error(t(strings.fund.sessionError));
      setOperatorSession(session);
      setLoginPassword("");
      setMfaChallenge("");
      setMfaCode("");
      setRecoveryCode("");
      setNotice(t(strings.fund.sessionSignedIn, { subject: session.subject || loginEmail.trim() }));
    } catch (reason) {
      setError(errorText(reason, t(strings.fund.sessionError)));
    } finally {
      setSessionBusy(false);
    }
  }

  async function signOut() {
    setSessionBusy(true);
    setError("");
    setNotice("");
    try {
      await signOutOperator();
      setOperatorSession({ authenticated: false });
      setLedgerDestination(undefined);
      setIntents([]);
      setActiveIntent(undefined);
      setHeadroom(undefined);
      setQuote(undefined);
      setWithdrawal(undefined);
      setNotice(t(strings.fund.sessionSignedOut));
    } catch (reason) {
      setError(errorText(reason, t(strings.fund.requestError)));
    } finally {
      setSessionBusy(false);
    }
  }

  async function createIntent(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError("");
    setNotice("");
    try {
      const created = await createFundingIntent({
        budgetId: form.budgetId.trim(),
        atomicAmount: toUSDCAtoms(form.amount, t(strings.fund.amountError)),
        valueMinor: toMinorUnits(form.value, t(strings.fund.valueError)),
      });
      setActiveIntent(created);
      setTransactionHash("");
      setNotice(t(strings.fund.createdNotice));
      await refreshIntents();
    } catch (reason) {
      setError(errorText(reason, t(strings.fund.requestError)));
    } finally {
      setBusy(false);
    }
  }

  async function confirmTransfer(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!activeIntent) return;
    setBusy(true);
    setError("");
    setNotice("");
    try {
      const updated = await confirmFundingTransfer(activeIntent.id, transactionHash.trim());
      setActiveIntent(updated);
      setNotice(updated.receiptDeliveryId
        ? t(strings.fund.acceptedNotice, { id: updated.receiptDeliveryId })
        : t(strings.fund.confirmedNotice, { state: updated.state }));
      await refreshIntents();
    } catch (reason) {
      setError(errorText(reason, t(strings.fund.requestError)));
    } finally {
      setBusy(false);
    }
  }

  async function payWithCoinbaseWallet() {
    const intent = activeIntent;
    const payDetails = intent && basePayDetails(intent);
    if (!intent || !payDetails) return;
    setBusy(true);
    setError("");
    setNotice("");
    try {
      const { pay, getPaymentStatus } = await import("@base-org/account");
      const payment = await pay({ amount: payDetails.amount, to: payDetails.address, testnet: false });
      if (!/^0x[0-9a-f]{64}$/i.test(payment.id)) throw new Error(t(strings.fund.requestError));
      setTransactionHash(payment.id);

      // Wallet status is only a progress hint. Treasury's configured Base RPC
      // remains the authority that verifies exact amount, recipient and finality.
      const status = await getPaymentStatus({ id: payment.id, testnet: false });
      if (status.status !== "completed") {
        setNotice(t(strings.fund.basePayPending, { status: status.status, hash: payment.id }));
        return;
      }
      if (status.amount !== payDetails.amount || status.recipient?.toLowerCase() !== payDetails.address.toLowerCase()) {
        setError(t(strings.fund.basePayMismatch, { hash: payment.id }));
        return;
      }

      const updated = await confirmFundingTransfer(intent.id, payment.id);
      setActiveIntent(updated);
      setNotice(updated.receiptDeliveryId
        ? t(strings.fund.acceptedNotice, { id: updated.receiptDeliveryId })
        : t(strings.fund.confirmedNotice, { state: updated.state }));
      await refreshHeadroom();
    } catch (reason) {
      setError(errorText(reason, t(strings.fund.requestError)));
    } finally {
      setBusy(false);
    }
  }

  async function requestQuote(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError("");
    setWithdrawal(undefined);
    try {
      const next = await previewWithdrawal({
        budgetId: form.budgetId.trim(),
        mandateId: mandateId.trim(),
        destination: destination.trim(),
        atomicAmount: toUSDCAtoms(withdrawalAmount, t(strings.fund.amountError)),
        valueMinor: toMinorUnits(withdrawalValue, t(strings.fund.valueError)),
        maxFeeWei: feeWei.trim(),
        feeValuationMinor: toMinorUnits(feeValue, t(strings.fund.valueError)),
      });
      setQuote(next);
    } catch (reason) {
      setError(errorText(reason, t(strings.fund.requestError)));
      setQuote(undefined);
    } finally {
      setBusy(false);
    }
  }

  async function runWithdrawal() {
    if (!quote || quote.executionMode !== "local-fork") return;
    setBusy(true);
    setError("");
    try {
      let idempotencyKey = withdrawalIdempotencyKeys.current.get(quote.id);
      if (!idempotencyKey) {
        idempotencyKey = crypto.randomUUID();
        withdrawalIdempotencyKeys.current.set(quote.id, idempotencyKey);
      }
      const result = await executeWithdrawal(quote.id, idempotencyKey);
      setWithdrawal(result);
      setNotice(t(strings.fund.withdrawNotice, { state: result.state }));
    } catch (reason) {
      setError(errorText(reason, t(strings.fund.requestError)));
    } finally {
      setBusy(false);
    }
  }

  async function copyPaymentUri() {
    if (!activeIntent?.paymentUri) return;
    try {
      await navigator.clipboard.writeText(activeIntent.paymentUri);
      setNotice(t(strings.fund.copiedNotice));
    } catch {
      setError(t(strings.fund.copyError));
    }
  }

  return (
    <section data-testid={selectors.pages.fund} aria-labelledby="fund-heading" className="mx-auto flex w-full max-w-6xl flex-col gap-6">
      <header className="flex items-start gap-3">
        <ArrowDownToLine aria-hidden="true" className="mt-1 size-7 shrink-0 text-app-primary" />
        <div>
          <h2 id="fund-heading" className="text-2xl font-semibold">{t(strings.fund.title)}</h2>
          <p className="mt-1 max-w-3xl text-app-muted-foreground">{t(strings.fund.description)}</p>
        </div>
      </header>

      <div role="note" className="flex gap-3 rounded-panel border border-app-warning/40 bg-app-warning/10 p-4 text-sm">
        <ShieldCheck aria-hidden="true" className="size-5 shrink-0 text-app-warning" />
        <p><strong>{t(strings.fund.forkTitle)}</strong> {t(strings.fund.forkHelp)}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t(strings.fund.onrampTitle)}</CardTitle>
          <p className="text-sm text-app-muted-foreground">{t(strings.fund.onrampIntro)}</p>
        </CardHeader>
        <CardContent className="grid gap-3 text-sm">
          <p>{t(strings.fund.onrampBuy)}</p>
          <p>{t(strings.fund.onrampTransfer)}</p>
          <p>{t(strings.fund.onrampGas)}</p>
          <p className="rounded-control border border-app-warning/40 bg-app-warning/10 p-3">{t(strings.fund.onrampSafety)}</p>
          <div className="flex flex-wrap gap-3">
            <a href="https://help.coinbase.com/en/wallet/managing-account/payment-method" target="_blank" rel="noreferrer" className="font-medium text-app-primary underline">{t(strings.fund.onrampBuyHelp)}</a>
            <a href="https://help.coinbase.com/en/wallet/managing-account/usdc-coinbase-wallet" target="_blank" rel="noreferrer" className="font-medium text-app-primary underline">{t(strings.fund.onrampUSDCFeesHelp)}</a>
            <a href="https://help.coinbase.com/en/wallet/sending-and-receiving/how-do-i-move-assets-between-my-coinbase-wallet-and-my-coinbase" target="_blank" rel="noreferrer" className="font-medium text-app-primary underline">{t(strings.fund.onrampTransferHelp)}</a>
          </div>
        </CardContent>
      </Card>

      <Card data-testid={selectors.funding.sessionPanel}>
        <CardHeader>
          <CardTitle>{t(strings.fund.sessionTitle)}</CardTitle>
          <p className="text-sm text-app-muted-foreground">{t(strings.fund.sessionHelp)}</p>
        </CardHeader>
        <CardContent>
          {sessionLoading ? <p role="status">{t(strings.fund.sessionLoading)}</p>
            : operatorSession.authenticated ? <div className="flex flex-wrap items-center justify-between gap-3">
              <p data-testid={selectors.funding.sessionIdentity} role="status">{t(strings.fund.sessionSignedIn, { subject: operatorSession.subject || "" })}</p>
              <Button data-testid={selectors.funding.signOutButton} type="button" variant="secondary" disabled={sessionBusy} onClick={() => void signOut()}>{t(strings.fund.signOut)}</Button>
            </div>
              : <form className="grid gap-4 sm:grid-cols-2" onSubmit={(event) => void signIn(event)}>
                <Field testId={selectors.funding.sessionEmail} label={t(strings.fund.sessionEmail)} value={loginEmail} type="email" onChange={setLoginEmail} autoComplete="email" required disabled={sessionBusy} />
                {!mfaChallenge && <Field testId={selectors.funding.sessionPassword} label={t(strings.fund.sessionPassword)} value={loginPassword} type="password" onChange={setLoginPassword} autoComplete="current-password" required disabled={sessionBusy} />}
                {mfaChallenge && <>
                  <Field testId={selectors.funding.sessionMfaCode} label={t(strings.fund.mfaCode)} value={mfaCode} onChange={setMfaCode} inputMode="numeric" autoComplete="one-time-code" disabled={sessionBusy} />
                  <Field testId={selectors.funding.sessionRecoveryCode} label={t(strings.fund.recoveryCode)} value={recoveryCode} onChange={setRecoveryCode} autoComplete="off" disabled={sessionBusy} />
                </>}
                <div className="flex flex-wrap gap-3 sm:col-span-2">
                  <Button data-testid={selectors.funding.signInButton} type="submit" disabled={sessionBusy || !loginEmail.trim() || (!mfaChallenge && !loginPassword)}>{sessionBusy ? t(strings.fund.signingIn) : t(mfaChallenge ? strings.fund.verifyMfa : strings.fund.signIn)}</Button>
                  {mfaChallenge && <Button type="button" variant="secondary" disabled={sessionBusy} onClick={() => { setMfaChallenge(""); setMfaCode(""); setRecoveryCode(""); setError(""); setNotice(""); }}>{t(strings.fund.cancelMfa)}</Button>}
                </div>
              </form>}
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>{t(strings.fund.budgetTitle)}</CardTitle><p className="text-sm text-app-muted-foreground">{t(strings.fund.budgetHelp)}</p></CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2">
          <Field testId={selectors.funding.budgetId} label={t(strings.fund.budgetId)} value={form.budgetId} onChange={(value) => setForm({ ...form, budgetId: value })} disabled={!operatorSession.authenticated} />
          <div className="sm:col-span-2 flex justify-end">
            <Button data-testid={selectors.funding.refreshButton} type="button" variant="secondary" disabled={busy || !operatorSession.authenticated || !form.budgetId.trim()} onClick={() => void refreshIntents()}>
              <RefreshCw aria-hidden="true" className="size-4" /> {t(strings.fund.refresh)}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card data-testid={selectors.funding.headroomCard}>
        <CardHeader><CardTitle>{t(strings.fund.headroomTitle)}</CardTitle><p className="text-sm text-app-muted-foreground">{t(strings.fund.headroomHelp)}</p></CardHeader>
        <CardContent>
          {headroom ? <dl className="grid gap-3 sm:grid-cols-3">
            <Detail testId={selectors.funding.creditedMinor} label={t(strings.fund.creditedReserve)} value={formatUSDMinor(headroom.fundingCreditedMinor, headroom.currency || "USD")} />
            <Detail testId={selectors.funding.remainingMinor} liveRegion label={t(strings.fund.remainingReserve)} value={formatUSDMinor(headroom.fundingRemainingMinor, headroom.currency || "USD")} />
            <Detail testId={selectors.funding.spendAvailableMinor} label={t(strings.fund.spendAvailability)} value={formatUSDMinor(headroom.availableMinor, headroom.currency || "USD")} />
          </dl> : <p data-testid={selectors.funding.headroomUnavailable} role={headroomError ? "alert" : "status"} className="text-sm text-app-muted-foreground">
            {headroomError || t(strings.fund.headroomUnavailable)}
          </p>}
        </CardContent>
      </Card>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader><CardTitle>{t(strings.fund.createTitle)}</CardTitle><p className="text-sm text-app-muted-foreground">{t(strings.fund.createHelp)}</p></CardHeader>
          <CardContent>
            <form className="grid gap-4" onSubmit={(event) => void createIntent(event)}>
              <Field testId={selectors.funding.atomicAmount} label={t(strings.fund.tokenAmount)} value={form.amount} onChange={(value) => setForm({ ...form, amount: value })} inputMode="decimal" pattern="[0-9]+(\.[0-9]{1,6})?" required />
              <Field testId={selectors.funding.declaredValue} label={t(strings.fund.declaredValue)} value={form.value} onChange={(value) => setForm({ ...form, value })} inputMode="decimal" pattern="[0-9]+(\.[0-9]{1,2})?" required />
              <div data-testid="funding-ledger-destination" className="sm:col-span-2 rounded-control border border-app-border p-3 text-sm" role={ledgerDestinationError ? "alert" : "status"}>
                <span className="font-medium">{t(strings.fund.ledgerDestinationLabel)}: </span>
                {ledgerDestination
                  ? `${ledgerDestination.bookName} · ${ledgerDestination.accountName} (${ledgerDestination.currency || "USD"})`
                  : ledgerDestinationError || t(operatorSession.authenticated ? strings.fund.ledgerDestinationLoading : strings.fund.ledgerDestinationConnect)}
              </div>
            <Button data-testid={selectors.funding.createIntentButton} type="submit" disabled={busy || !operatorSession.authenticated || !ledgerDestination || !form.budgetId.trim()}>{t(strings.fund.createLink)}</Button>
            </form>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle>{t(strings.fund.statusTitle)}</CardTitle><p className="text-sm text-app-muted-foreground">{t(strings.fund.statusHelp)}</p></CardHeader>
          <CardContent className="grid gap-4">
            {activeIntent ? <div className="rounded-control border border-app-border p-3 text-sm">
              <dl className="grid gap-2 sm:grid-cols-2">
                <Detail testId={selectors.funding.activeIntentState} liveRegion label={t(strings.fund.state)} value={activeIntent.state} />
                <Detail label={t(strings.fund.tokenAmount)} value={formatAtomicUSDC(activeIntent.atomicAmount)} />
                <Detail testId={selectors.funding.notificationState} liveRegion label={t(strings.fund.notification)} value={activeIntent.notificationState || t(strings.fund.notAvailable)} />
              </dl>
              {activeIntent.paymentUri && <div className="mt-4 grid gap-4 sm:grid-cols-[auto_minmax(0,1fr)] sm:items-center">
                <figure className="grid justify-items-center gap-2 rounded-control border border-app-border bg-app-card p-3">
                  <QRCodeSVG data-testid={selectors.funding.paymentQr} role="img" aria-label={t(strings.fund.paymentQrLabel)} value={activeIntent.paymentUri} size={216} level="M" marginSize={1} />
                  <figcaption className="max-w-56 text-center text-xs text-app-muted-foreground">{t(strings.fund.paymentQrHelp)}</figcaption>
                </figure>
                <div className="grid min-w-0 gap-2">
                  <p className="font-medium">{t(strings.fund.paymentLink)}</p>
                  <p data-testid={selectors.funding.paymentUri} className="text-sm text-app-muted-foreground">{t(strings.fund.paymentLinkHelp)}</p>
                  {basePayDetails(activeIntent) && <div className="grid gap-2 rounded-control border border-app-border p-3">
                    <p className="text-sm text-app-muted-foreground">{t(strings.fund.basePayHelp)}</p>
                    <Button data-testid={selectors.funding.basePayButton} type="button" disabled={busy || !operatorSession.authenticated} onClick={() => void payWithCoinbaseWallet()}>{t(strings.fund.basePayButton)}</Button>
                  </div>}
                  <div className="flex flex-wrap gap-2">
                    <Button data-testid={selectors.funding.copyLinkButton} type="button" variant="secondary" onClick={() => void copyPaymentUri()}><Clipboard aria-hidden="true" className="size-4" /> {t(strings.fund.copyLink)}</Button>
                    {activeIntent.paymentUri.startsWith("ethereum:") && <a data-testid={selectors.funding.openWalletLink} href={activeIntent.paymentUri} className="inline-flex min-h-10 items-center gap-2 rounded-control border border-app-border px-3 text-sm font-medium text-app-foreground hover:bg-app-muted"><ExternalLink aria-hidden="true" className="size-4" /> {t(strings.fund.openWallet)}</a>}
                  </div>
                </div>
              </div>}
              <form className="mt-4 grid gap-3 border-t border-app-border pt-4" onSubmit={(event) => void confirmTransfer(event)}>
                <Field testId={selectors.funding.transactionHash} label={t(strings.fund.transactionHash)} value={transactionHash} onChange={setTransactionHash} required />
                <Button data-testid={selectors.funding.confirmTransferButton} type="submit" variant="secondary" disabled={busy || !operatorSession.authenticated}>{t(strings.fund.confirmAllocate)}</Button>
                <p className="text-xs text-app-muted-foreground">{t(strings.fund.confirmationHelp)}</p>
              </form>
            </div> : <p className="rounded-control border border-dashed border-app-border p-5 text-sm text-app-muted-foreground">{t(strings.fund.emptyIntent)}</p>}
            {intents.length > 0 && <div><h3 className="mb-2 font-semibold">{t(strings.fund.recentIntents)}</h3><ul className="grid gap-2">{intents.map((item) => <li key={item.id}><button data-testid={selectors.funding.recentIntent} type="button" className="w-full rounded-control border border-app-border p-3 text-start hover:bg-app-muted" onClick={() => setActiveIntent(item)}><span className="block text-sm">{t(strings.fund.atomicUnits, { state: item.state, amount: formatAtomicUSDC(item.atomicAmount) })}</span></button></li>)}</ul></div>}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader><CardTitle><span className="inline-flex items-center gap-2"><ArrowUpFromLine aria-hidden="true" className="size-5" /> {t(strings.fund.withdrawTitle)}</span></CardTitle><p className="text-sm text-app-muted-foreground">{t(strings.fund.withdrawHelp)}</p></CardHeader>
        <CardContent>
          <form className="grid gap-4 sm:grid-cols-2" onSubmit={(event) => void requestQuote(event)}>
            <Field testId={selectors.funding.withdrawalDestination} label={t(strings.fund.destination)} value={destination} onChange={setDestination} required />
            <Field testId={selectors.funding.mandateId} label={t(strings.fund.mandateId)} value={mandateId} onChange={setMandateId} required />
            <Field testId={selectors.funding.withdrawalAmount} label={t(strings.fund.tokenAmount)} value={withdrawalAmount} onChange={setWithdrawalAmount} inputMode="decimal" pattern="[0-9]+(\.[0-9]{1,6})?" required />
            <Field testId={selectors.funding.withdrawalValue} label={t(strings.fund.declaredValue)} value={withdrawalValue} onChange={setWithdrawalValue} inputMode="decimal" pattern="[0-9]+(\.[0-9]{1,2})?" required />
            <Field testId={selectors.funding.withdrawalMaxFee} label={t(strings.fund.maxFee)} value={feeWei} onChange={setFeeWei} inputMode="numeric" pattern="[0-9]+" required />
            <Field testId={selectors.funding.withdrawalFeeValue} label={t(strings.fund.feeValue)} value={feeValue} onChange={setFeeValue} inputMode="decimal" pattern="[0-9]+(\.[0-9]{1,2})?" required />
            <div className="sm:col-span-2"><Button data-testid={selectors.funding.withdrawalPreviewButton} type="submit" variant="secondary" disabled={busy || !operatorSession.authenticated || !form.budgetId.trim() || !mandateId.trim()}>{t(strings.fund.preview)}</Button></div>
          </form>
          {quote && <div className="mt-5 rounded-control border border-app-border p-4">
            <h3 className="font-semibold">{t(strings.fund.quote)}</h3>
            <dl className="mt-3 grid gap-2 text-sm sm:grid-cols-3">
              <Detail label={t(strings.fund.network)} value={quote.network} />
              <Detail label={t(strings.fund.executionMode)} value={quote.executionMode} />
              <Detail label={t(strings.fund.amountValue)} value={formatAtomicUSDC(quote.atomicAmount)} />
              <Detail label={t(strings.fund.destination)} value={quote.destination} />
              <Detail label={t(strings.fund.feeEstimate)} value={formatUSDMinor(quote.feeValuationMinor, quote.currency)} />
              <Detail label={t(strings.fund.estimatedTime)} value={formatDuration(quote.estimatedSeconds)} />
              <Detail label={t(strings.fund.quoteExpires)} value={formatTimestamp(quote.expiresAt?.seconds)} />
            </dl>
            <Button data-testid={selectors.funding.withdrawalExecuteButton} type="button" className="mt-4" variant="danger" disabled={busy || !operatorSession.authenticated || quote.executionMode !== "local-fork" || withdrawal?.state === "pending_approval"} onClick={() => void runWithdrawal()}>
              {quote.executionMode === "local-fork" ? t(strings.fund.executeFork) : t(strings.fund.executeDisabled)}
            </Button>
            {withdrawal && <p data-testid={selectors.funding.withdrawalStatus} role="status" className="mt-3 text-sm">{t(strings.fund.withdrawalStatus, { id: withdrawal.id, state: withdrawal.state })}{withdrawal.transactionHash ? ` · ${withdrawal.transactionHash}` : ""}</p>}
          </div>}
        </CardContent>
      </Card>
      {error && <p role="alert" className="rounded-control border border-app-danger/40 bg-app-danger/10 p-3 text-sm font-medium text-app-danger">{error}</p>}
      {notice && <p role="status" className="rounded-control border border-app-success/40 bg-app-success/10 p-3 text-sm">{notice}</p>}
    </section>
  );
}

function Field({ label, value, onChange, type = "text", testId, ...props }: { label: string; value: string; onChange: (value: string) => void; type?: string; testId?: string; autoComplete?: string; inputMode?: "text" | "numeric" | "decimal"; pattern?: string; required?: boolean; disabled?: boolean }) {
  const id = useId();
  return <div className="grid min-w-0 gap-2"><label htmlFor={id} className="text-sm font-medium">{label}</label><Input id={id} data-testid={testId} type={type} value={value} onChange={(event) => onChange(event.target.value)} {...props} /></div>;
}

function Detail({ label, value, testId, liveRegion = false }: { label: string; value: string; testId?: string; liveRegion?: boolean }) {
  return <div className="min-w-0"><dt className="text-app-muted-foreground">{label}</dt><dd data-testid={testId} aria-live={liveRegion ? "polite" : undefined} className="break-all font-medium">{value || "—"}</dd></div>;
}

function toMinorUnits(value: string, errorMessage: string): bigint {
  const match = /^(\d+)(?:\.(\d{1,2}))?$/.exec(value.trim());
  if (!match) throw new Error(errorMessage);
  return BigInt(match[1] ?? "0") * 100n + BigInt((match[2] ?? "").padEnd(2, "0") || "0");
}

function toUSDCAtoms(value: string, errorMessage: string): string {
  const match = /^(\d+)(?:\.(\d{1,6}))?$/.exec(value.trim());
  if (!match) throw new Error(errorMessage);
  const whole = BigInt(match[1] ?? "0");
  const fraction = BigInt((match[2] ?? "").padEnd(6, "0") || "0");
  const atoms = whole * 1_000_000n + fraction;
  if (atoms <= 0n) throw new Error(errorMessage);
  return atoms.toString();
}

function errorText(reason: unknown, fallback: string): string {
  return reason instanceof Error ? reason.message : fallback;
}

function formatTimestamp(seconds: bigint | undefined): string {
  if (seconds === undefined) return "—";
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(Number(seconds) * 1000);
}

function formatAtomicUSDC(value: string): string {
  if (!/^\d+$/.test(value)) return value;
  const atoms = BigInt(value);
  const whole = atoms / 1_000_000n;
  const fraction = (atoms % 1_000_000n).toString().padStart(6, "0");
  return `${whole}.${fraction} USDC`;
}

function formatUSDMinor(value: bigint, currency: string): string {
  const magnitude = value < 0n ? -value : value;
  const whole = magnitude / 100n;
  const fraction = (magnitude % 100n).toString().padStart(2, "0");
  if (magnitude <= BigInt(Number.MAX_SAFE_INTEGER)) {
    return new Intl.NumberFormat(undefined, { style: "currency", currency, minimumFractionDigits: 2 }).format(Number(value) / 100);
  }
  return `${value < 0n ? "-" : ""}${currency} ${whole}.${fraction}`;
}

function formatDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return "—";
  if (seconds < 60) return `${Math.ceil(seconds)} sec`;
  const minutes = Math.ceil(seconds / 60);
  return `${minutes} min`;
}

function isFundingIntentPending(state: string): boolean {
  return state === "waiting" || state === "observed" || state === "delivering" || state === "awaiting_transfer";
}

function basePayDetails(intent: FundingIntent): { amount: string; address: string } | undefined {
  if (intent.fundingMode !== "base-mainnet" || intent.paymentCurrency.toUpperCase() !== "USD" || !isFundingIntentPending(intent.state)) return undefined;
  const address = intent.wallet?.address ?? "";
  if (!/^0x[0-9a-f]{40}$/i.test(address) || !/^\d+$/.test(intent.atomicAmount)) return undefined;
  let valueMinor: bigint;
  let atomicAmount: bigint;
  try {
    valueMinor = BigInt(intent.paymentValueMinor);
    atomicAmount = BigInt(intent.atomicAmount);
  } catch {
    return undefined;
  }
  if (valueMinor <= 0n || atomicAmount !== valueMinor * 10_000n) return undefined;
  return { amount: formatUSDMinorForPayment(valueMinor), address };
}

function formatUSDMinorForPayment(valueMinor: bigint): string {
  return `${valueMinor / 100n}.${(valueMinor % 100n).toString().padStart(2, "0")}`;
}
