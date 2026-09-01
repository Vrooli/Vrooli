import { FormEvent, useState } from "react";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";

export function AgentsPage() {
  const { t } = useTranslation();
  const [description, setDescription] = useState("");
  const [draft, setDraft] = useState<string>();
  const [confirmed, setConfirmed] = useState(false);
  const submit = (event: FormEvent) => {
    event.preventDefault();
    const value = description.trim();
    if (value) setDraft(value);
  };
  return <section aria-labelledby="agents-heading" className="flex flex-col gap-4">
    <h2 id="agents-heading" className="text-2xl font-semibold">{t(strings.console.agents.title)}</h2>
    <p className="text-app-muted-foreground">{t(strings.console.agents.description)}</p>
    <ExperienceSurface surfaceId="roster-region" state="empty" className="rounded-lg border p-6">
    <ExperienceSurface surfaceId="draft-region" state="ready">
      <form onSubmit={submit}>
      <h3 className="font-semibold">{t(strings.console.agents.draftHeading)}</h3>
      <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.console.agents.draftDescription)}</p>
      <label className="mt-4 flex flex-col gap-1 text-sm" htmlFor="agent-description">{t(strings.console.agents.descriptionLabel)}
        <textarea id="agent-description" value={description} onChange={(event) => setDescription(event.target.value)} className="min-h-24 rounded border p-2" placeholder={t(strings.console.agents.descriptionPlaceholder)} />
      </label>
      <button type="submit" className="mt-4 rounded bg-primary px-3 py-2 text-primary-foreground">{t(strings.console.agents.prepareDraft)}</button>
      </form>
    </ExperienceSurface>
    </ExperienceSurface>
    {draft && <section aria-live="polite" className="rounded-lg border p-6">
      <h3 className="font-semibold">{t(strings.console.agents.reviewHeading)}</h3>
      <p className="mt-2">{draft}</p>
      <p className="mt-2 text-sm text-app-muted-foreground">{t(strings.console.agents.defaultGrant)}</p>
      {!confirmed ? <button type="button" className="mt-4 rounded bg-primary px-3 py-2 text-primary-foreground" onClick={() => setConfirmed(true)}>{t(strings.console.agents.confirmWrite)}</button> : <p className="mt-4 text-sm">{t(strings.console.agents.confirmed)}</p>}
    </section>}
  </section>;
}
