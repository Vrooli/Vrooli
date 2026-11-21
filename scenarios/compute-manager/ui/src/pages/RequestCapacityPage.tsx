import { FormEvent, useState } from "react";
import { Link } from "react-router-dom";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "@vrooli/react-component-library/Input/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";

import { requestInstance } from "../api/compute";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

export function RequestCapacityPage() {
  const { t } = useTranslation();
  const [region, setRegion] = useState("fsn1");
  const [size, setSize] = useState("cx22");
  const [lifetime, setLifetime] = useState("3600");
  const [state, setState] = useState<"idle" | "submitting" | "success" | "error" | "validation">("idle");

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const lifetimeSeconds = Number(lifetime);
    if (!Number.isFinite(lifetimeSeconds) || lifetimeSeconds <= 0) {
      setState("validation");
      return;
    }
    setState("submitting");
    requestInstance({
      idempotencyKey: `ui-${Date.now()}`,
      provider: "hetzner",
      region,
      size,
      lifetimeSeconds: BigInt(Math.floor(lifetimeSeconds)),
    }).then(() => setState("success"), () => setState("error"));
  };

  return (
    <section data-testid={selectors.pages.request} aria-labelledby="request-heading" className="flex flex-col gap-space-md">
      <PageHeader headingId="request-heading" title={t(strings.pages.request.title)} description={t(strings.pages.request.description)} />
      <form data-testid={selectors.pages.requestForm} onSubmit={submit} className="grid max-w-2xl gap-space-md rounded-card border border-border bg-surface p-space-md">
        <label className="grid gap-space-xs" htmlFor="request-provider">{t(strings.pages.request.provider)}<Input id="request-provider" value="hetzner" readOnly /></label>
        <label className="grid gap-space-xs" htmlFor="request-region">{t(strings.pages.request.region)}<Input id="request-region" value={region} onChange={(event) => setRegion(event.target.value)} /></label>
        <label className="grid gap-space-xs" htmlFor="request-size">{t(strings.pages.request.size)}<Input id="request-size" value={size} onChange={(event) => setSize(event.target.value)} /></label>
        <label className="grid gap-space-xs" htmlFor="request-lifetime">{t(strings.pages.request.lifetime)}<Input id="request-lifetime" type="number" min="1" value={lifetime} onChange={(event) => setLifetime(event.target.value)} /></label>
        <p data-testid="cost-estimate" role="status" className="text-muted">{t(strings.pages.request.estimate)}</p>
        <Button data-testid={selectors.pages.requestSubmit} type="submit" disabled={state === "submitting"} pending={state === "submitting"} pendingLabel={t(strings.pages.request.submitting)}>{t(strings.pages.request.submit)}</Button>
        <div data-testid={selectors.pages.requestStatus} aria-live="polite">
          {state === "success" && <p role="status">{t(strings.pages.request.success)} <Link to="/">{t(strings.pages.dashboard.inventoryTitle)}</Link></p>}
          {state === "error" && <p role="alert">{t(strings.pages.request.error)}</p>}
          {state === "validation" && <p role="alert">{t(strings.pages.request.validation)}</p>}
        </div>
      </form>
    </section>
  );
}
