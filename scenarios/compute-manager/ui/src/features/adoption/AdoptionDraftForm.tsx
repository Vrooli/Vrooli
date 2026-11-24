import { useState, type FormEvent } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "@vrooli/react-component-library/Input/1";

import { selectors } from "../../consts/selectors";
import { buildAdoptionDraft, type AdoptionDraft, type AdoptionDraftError } from "./model";

export interface AdoptionDraftFormCopy {
  intro: string;
  displayName: string;
  host: string;
  user: string;
  monthlyCostMinor: string;
  currency: string;
  taxBasis: string;
  continueLabel: string;
  errors: Record<AdoptionDraftError, string>;
}

export function AdoptionDraftForm({ onSubmit, copy }: { onSubmit: (draft: AdoptionDraft) => void; copy: AdoptionDraftFormCopy }) {
  const [displayName, setDisplayName] = useState("");
  const [host, setHost] = useState("");
  const [user, setUser] = useState("");
  const [monthlyCostMinor, setMonthlyCostMinor] = useState("");
  const [currency, setCurrency] = useState("");
  const [taxBasis, setTaxBasis] = useState("");
  const [error, setError] = useState<AdoptionDraftError>();

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = buildAdoptionDraft({ displayName, host, user, monthlyCostMinor, currency, taxBasis });
    if (!result.ok) {
      setError(result.error);
      return;
    }
    setError(undefined);
    onSubmit(result.value);
  };

  return (
    <form data-testid={selectors.adoption.form} onSubmit={submit} className="grid max-w-2xl gap-space-md rounded-card border border-border bg-surface p-space-md">
      <p className="text-sm text-muted">{copy.intro}</p>
      <label className="grid gap-space-xs" htmlFor={selectors.adoption.displayName}>{copy.displayName}<Input id={selectors.adoption.displayName} value={displayName} onChange={(event) => setDisplayName(event.target.value)} /></label>
      <label className="grid gap-space-xs" htmlFor={selectors.adoption.host}>{copy.host}<Input id={selectors.adoption.host} autoComplete="url" value={host} onChange={(event) => setHost(event.target.value)} /></label>
      <label className="grid gap-space-xs" htmlFor={selectors.adoption.user}>{copy.user}<Input id={selectors.adoption.user} autoComplete="username" value={user} onChange={(event) => setUser(event.target.value)} /></label>
      <label className="grid gap-space-xs" htmlFor={selectors.adoption.monthlyCost}>{copy.monthlyCostMinor}<Input id={selectors.adoption.monthlyCost} type="number" min="0" step="1" inputMode="numeric" value={monthlyCostMinor} onChange={(event) => setMonthlyCostMinor(event.target.value)} /></label>
      <label className="grid gap-space-xs" htmlFor={selectors.adoption.currency}>{copy.currency}<Input id={selectors.adoption.currency} autoComplete="currency" value={currency} onChange={(event) => setCurrency(event.target.value)} /></label>
      <label className="grid gap-space-xs" htmlFor={selectors.adoption.taxBasis}>{copy.taxBasis}<Input id={selectors.adoption.taxBasis} value={taxBasis} onChange={(event) => setTaxBasis(event.target.value)} /></label>
      {error && <p role="alert">{copy.errors[error]}</p>}
      <Button type="submit" data-testid={selectors.adoption.submit}>{copy.continueLabel}</Button>
    </form>
  );
}
