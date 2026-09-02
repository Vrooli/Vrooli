import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Check, Lock } from "lucide-react";
import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";

import { Button } from "@vrooli/react-component-library/Button/2";

import { ConsoleApiError, consoleApi, consoleKeys, type AgentDraft } from "../api/console";
import { AgentMark } from "../components/console/AgentMark";
import { Page } from "../components/console/Page";
import { Region } from "../components/console/Region";
import { strings } from "../consts/strings";
import { useSession } from "../features/session/SessionProvider";
import { useTranslation } from "../i18n";

/**
 * Assisted agent authoring: describe the job, review the typed draft, confirm.
 * Nothing is written until the confirm step, and owner-only capabilities are
 * never proposed by the draft.
 */
export function AgentNewPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { withSession } = useSession();
  const [description, setDescription] = useState("");
  const [draft, setDraft] = useState<AgentDraft>();

  const prepare = useMutation({ mutationFn: (text: string) => consoleApi.draftAgent(text), onSuccess: setDraft });
  const confirm = useMutation({
    mutationFn: (value: AgentDraft) => withSession(() => consoleApi.createAgent(value)),
    onSuccess: async (created) => {
      await queryClient.invalidateQueries({ queryKey: consoleKeys.agents });
      navigate(`/agents/${encodeURIComponent(created.id)}`);
    },
  });

  const submit = (event: FormEvent) => {
    event.preventDefault();
    const value = description.trim();
    if (value) prepare.mutate(value);
  };

  const state = prepare.isPending || confirm.isPending ? "loading" : prepare.isError ? "error" : "ready";
  const notWritable = confirm.error instanceof ConsoleApiError && confirm.error.status === 501;

  return (
    <Page
      headingId="agent-new-heading"
      testId="page-agent-new"
      eyebrow={
        <Link to="/agents" className="inline-flex items-center gap-1 text-app-primary">
          <ArrowLeft aria-hidden="true" className="h-3 w-3" />
          {t(strings.console.agents.title)}
        </Link>
      }
      title={t(strings.console.agents.newAgent)}
      description={t(strings.console.agents.draftDescription)}
    >
      <Region surfaceId="draft-region" testId="agent-new-draft-region" state={state} errorDetail={prepare.error instanceof Error ? prepare.error.message : undefined} onRetry={() => prepare.reset()}>
        <div className="grid gap-6 lg:grid-cols-2">
          <form onSubmit={submit} className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4">
            <label className="flex flex-col gap-1.5 text-sm font-medium" htmlFor="agent-description">
              {t(strings.console.agents.descriptionLabel)}
              <textarea
                id="agent-description"
                data-testid="agent-new-description"
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                rows={5}
                className="rounded-control border border-app-border bg-app-surface p-3 text-base font-normal text-app-foreground placeholder:text-app-muted-foreground md:text-sm"
                placeholder={t(strings.console.agents.descriptionPlaceholder)}
              />
            </label>
            <p className="text-xs text-app-muted-foreground">{t(strings.console.agents.draftHint)}</p>
            <div>
              <Button type="submit" data-testid="agent-new-prepare" disabled={!description.trim()} pending={prepare.isPending}>
                {t(strings.console.agents.prepareDraft)}
              </Button>
            </div>
          </form>

          <section aria-live="polite" aria-label={t(strings.console.agents.reviewHeading)} className="flex flex-col gap-3 rounded-panel border border-dashed border-app-border p-4">
            <h3 className="text-sm font-semibold">{t(strings.console.agents.reviewHeading)}</h3>
            {draft ? (
              <>
                <div className="flex items-center gap-3">
                  <AgentMark name={draft.display_name} size="lg" />
                  <label className="flex min-w-0 flex-1 flex-col gap-1 text-xs font-medium text-app-muted-foreground">
                    {t(strings.console.agents.displayName)}
                    <input
                      data-testid="agent-new-name"
                      value={draft.display_name}
                      onChange={(event) => setDraft({ ...draft, display_name: event.target.value })}
                      className="min-h-10 rounded-control border border-app-border bg-app-surface px-3 text-sm font-medium text-app-foreground"
                    />
                  </label>
                </div>
                <p className="text-sm text-app-foreground">{draft.description}</p>
                <div className="flex flex-wrap items-center gap-1.5 text-xs">
                  <span className="text-app-muted-foreground">{t(strings.console.agents.proposedGrant)}</span>
                  {draft.scopes.map((scope) => (
                    <code key={scope} className="rounded-sm bg-app-surface-muted px-1.5 py-0.5 font-mono">
                      {scope}
                    </code>
                  ))}
                </div>
                <p className="flex items-start gap-2 text-xs text-app-muted-foreground">
                  <Lock aria-hidden="true" className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                  {t(strings.console.agents.defaultGrant)}
                </p>
                <div className="flex flex-wrap items-center gap-2 pt-1">
                  <Button type="button" data-testid="agent-new-confirm" pending={confirm.isPending} onClick={() => confirm.mutate(draft)}>
                    <Check aria-hidden="true" className="h-4 w-4" />
                    {t(strings.console.agents.confirmWrite)}
                  </Button>
                  <Button type="button" variant="ghost" onClick={() => setDraft(undefined)}>
                    {t(strings.console.common.cancel)}
                  </Button>
                </div>
                {confirm.isError ? (
                  <p role="alert" className="rounded-control border border-app-danger/40 bg-app-danger/5 px-3 py-2 text-xs text-app-foreground">
                    {notWritable ? t(strings.console.agents.notWritable) : confirm.error instanceof Error ? confirm.error.message : t(strings.errors.unknown)}
                  </p>
                ) : null}
              </>
            ) : (
              <p className="text-sm text-app-muted-foreground">{t(strings.console.agents.reviewPlaceholder)}</p>
            )}
          </section>
        </div>
      </Region>
    </Page>
  );
}
