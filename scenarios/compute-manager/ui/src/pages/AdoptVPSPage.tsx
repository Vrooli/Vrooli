import { useCallback, useState } from "react";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";

import { adoptVPS, fetchSpendReceipt } from "../api/compute";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { AdoptionFlow, type AdoptionFlowCopy } from "../features/adoption/AdoptionFlow";
import type { AdoptionDraft, AdoptionRecordSnapshot } from "../features/adoption/model";

const REQUEST_KEY_STORAGE = "compute-manager.pending-adopt-vps";

export function AdoptVPSPage() {
  const { t, i18n } = useTranslation();
  const [requestKey] = useState(() => sessionStorage.getItem(REQUEST_KEY_STORAGE) ?? `adopt-${crypto.randomUUID()}`);
  const [receipt, setReceipt] = useState<Awaited<ReturnType<typeof adoptVPS>>["spendReceipt"]>();

  const createAdoption = useCallback(async (draft: AdoptionDraft): Promise<AdoptionRecordSnapshot> => {
    sessionStorage.setItem(REQUEST_KEY_STORAGE, requestKey);
    let response: Awaited<ReturnType<typeof adoptVPS>>;
    try {
      response = await adoptVPS({
        idempotencyKey: requestKey,
        host: draft.host,
        user: draft.user,
        displayName: draft.displayName,
        monthlyCostMinor: BigInt(draft.monthlyCostMinor),
        currency: draft.currency,
        taxBasis: draft.taxBasis,
      });
    } catch {
      // Admission refusal still persists a quote receipt. Read it back by the
      // stable request key so the user can inspect the cap decision safely.
      try {
        const stored = await fetchSpendReceipt(requestKey);
        setReceipt(stored.receipt);
      } catch { /* the flow shows a generic retry-safe message */ }
      throw new Error("adoption request did not complete");
    }
    const instance = response.instance;
    const spendReceipt = response.spendReceipt;
    setReceipt(spendReceipt);
    return {
      id: instance?.id ?? requestKey,
      displayName: instance?.displayName,
      credentialStepRef: instance?.credentialStepRef,
      host: instance?.address ?? draft.host,
      user: instance?.sshUser ?? draft.user,
      monthlyCostMinor: spendReceipt ? Number(spendReceipt.totalMinorPerMonth) : draft.monthlyCostMinor,
      currency: spendReceipt?.currency ?? draft.currency,
      taxBasis: spendReceipt?.taxBasis ?? draft.taxBasis,
      state: (instance?.adoptionState || "awaiting_secret") as AdoptionRecordSnapshot["state"],
      adoptionEvidenceVerified: instance?.adoptionEvidenceVerified ?? false,
    };
  }, [requestKey]);

  const copy: AdoptionFlowCopy = {
    draftForm: {
      intro: t(strings.pages.adoption.intro),
      displayName: t(strings.pages.adoption.displayName),
      host: t(strings.pages.adoption.host),
      user: t(strings.pages.adoption.user),
      monthlyCostMinor: t(strings.pages.adoption.monthlyCostMinor),
      currency: t(strings.pages.adoption.currency),
      taxBasis: t(strings.pages.adoption.taxBasis),
      continueLabel: t(strings.pages.adoption.continueLabel),
      errors: {
        address_required: t(strings.pages.adoption.addressRequired),
        ssh_user_required: t(strings.pages.adoption.sshUserRequired),
        monthly_cost_invalid: t(strings.pages.adoption.monthlyCostInvalid),
        currency_invalid: t(strings.pages.adoption.currencyInvalid),
        tax_basis_required: t(strings.pages.adoption.taxBasisRequired),
      },
    },
    statusCard: {
      stateLabel: t(strings.pages.adoption.stateLabel),
      states: {
        draft: t(strings.pages.adoption.stateDraft),
        awaiting_secret: t(strings.pages.adoption.stateAwaitingSecret),
        credential_stored: t(strings.pages.adoption.stateCredentialStored),
        bridge_enrolling: t(strings.pages.adoption.stateBridgeEnrolling),
        ready: t(strings.pages.adoption.stateReady),
        failed: t(strings.pages.adoption.stateFailed),
        released: t(strings.pages.adoption.stateReleased),
      },
      host: t(strings.pages.adoption.host),
      user: t(strings.pages.adoption.user),
      monthlyCost: t(strings.pages.adoption.monthlyCost),
      taxBasis: t(strings.pages.adoption.taxBasis),
      releaseSemantics: t(strings.pages.adoption.releaseSemantics),
      verificationPending: t(strings.pages.adoption.verificationPending),
      minorUnitsLabel: t(strings.pages.adoption.minorUnits),
      locale: i18n.language,
    },
    createFailed: t(strings.pages.adoption.createFailed),
    credentialStepUnavailable: t(strings.pages.adoption.credentialStepUnavailable),
    credentialStepAction: t(strings.pages.adoption.credentialStepAction),
  };

  return <section className="flex flex-col gap-space-md">
    <PageHeader headingId="adoption-heading" title={t(strings.pages.adoption.title)} description={t(strings.pages.adoption.description)} />
    <AdoptionFlow createAdoption={createAdoption} onOpenCredentialStep={(ref) => { window.location.assign(ref); }} copy={copy} />
    {receipt && <article className="grid max-w-2xl gap-2 rounded-card border border-border bg-surface p-space-md" aria-label={t(strings.pages.adoption.quote)}>
      <h2 className="text-lg font-semibold">{t(strings.pages.adoption.quote)}</h2>
      <p>{t(strings.pages.adoption.admission)}: <strong>{receipt.admissionDecision}</strong> · {receipt.totalMinorPerMonth.toString()} {receipt.currency} / {t(strings.pages.adoption.monthlyCost)} ({t(strings.pages.adoption.minorUnits)})</p>
      <p>{t(strings.pages.adoption.taxBasis)}: {receipt.taxBasis} · {t(strings.pages.adoption.cap)}: {receipt.capMonthlyMinor.toString()} {receipt.capCurrency} ({t(strings.pages.adoption.minorUnits)})</p>
      <p>{t(strings.pages.adoption.source)}: {receipt.sourceUrl || t(strings.pages.adoption.sourceUnknown)} · {receipt.sourceVersion || t(strings.pages.adoption.sourceUnknown)}</p>
      <p>{t(strings.pages.adoption.observation)}: {formatTimestamp(receipt.observedAt) ?? t(strings.pages.adoption.sourceUnknown)} · {t(strings.pages.adoption.expires)}: {formatTimestamp(receipt.expiresAt) ?? t(strings.pages.adoption.sourceUnknown)}</p>
    </article>}
  </section>;
}

function formatTimestamp(value?: { seconds: bigint; nanos: number }) {
  return value ? new Date(Number(value.seconds) * 1000 + value.nanos / 1_000_000).toLocaleString() : undefined;
}
