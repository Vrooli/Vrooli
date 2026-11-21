import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Loader2, Pencil } from "lucide-react";

import { Panel } from "../../components/ui/panel";
import { Table, TBody, TD, TH, THead, TR } from "../../components/ui/table";
import { StatusDot } from "../../components/ui/status-dot";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { pushToast } from "../../components/ui/toast";
import { PageHeader } from "../../components/composites/PageHeader";
import { ApiErrorState } from "../../components/composites/ApiErrorState";
import { LoadingRows } from "../../components/composites/LoadingRows";
import { getVoiceOverrides, setVoiceOverride, type VoiceOverrideRow } from "../../services/settings";
import { getStatus } from "../../services/tts";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { TTS_ADAPTERS } from "../providers/providerCatalog.generated";

const CANONICAL_VOICES = [
  "voice.feminine.warm",
  "voice.feminine.neutral",
  "voice.masculine.warm",
  "voice.masculine.neutral",
  "voice.neutral.default",
];

interface EditTarget {
  canonicalVoice: string;
  tierProvider: string;
}

export function VoicesPage() {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const overrides = useQuery({ queryKey: ["settings", "voices"], queryFn: getVoiceOverrides });
  const ttsStatus = useQuery({ queryKey: ["tts", "status"], queryFn: getStatus, refetchInterval: 30_000 });

  // A single edit target keeps the matrix unambiguous: starting an edit on a
  // second cell replaces the first editor instead of opening two at once.
  const [editing, setEditing] = useState<EditTarget | null>(null);
  const [draft, setDraft] = useState("");
  const [saveError, setSaveError] = useState<string | null>(null);
  const editorRef = useRef<HTMLInputElement>(null);

  // The cell that opens the editor removes its own trigger, so move focus into
  // the input explicitly (autofocus is disallowed by the a11y lint rule).
  useEffect(() => {
    if (editing) editorRef.current?.focus();
  }, [editing]);

  const saveOverride = useMutation({
    mutationFn: (vars: EditTarget & { adapterVoice: string }) =>
      setVoiceOverride(vars.canonicalVoice, vars.tierProvider, vars.adapterVoice),
    onSuccess: (result, vars) => {
      if (!result.ok) {
        setSaveError(t(strings.voices.saveError));
        pushToast({
          title: t(strings.voices.saveError),
          body: result.error.message,
        });
        return;
      }
      qc.setQueryData<typeof result>(["settings", "voices"], result);
      setEditing(null);
      setDraft("");
      setSaveError(null);
      pushToast({
        title:
          vars.adapterVoice === ""
            ? t(strings.voices.deleted)
            : t(strings.voices.saved),
      });
    },
    onError: (error) => {
      setSaveError(t(strings.voices.saveError));
      pushToast({
        title: t(strings.voices.saveError),
        body: error instanceof Error ? error.message : String(error),
      });
    },
  });

  const probes = ttsStatus.data?.ok
    ? ttsStatus.data.data.availability.map((a) => ({
        id: a.providerId || a.tier,
        role: a.tier.toUpperCase(),
        tone: a.available ? ("success" as const) : ("danger" as const),
        label: a.available ? t(strings.status.ok) : t(strings.status.offline),
      }))
    : [];

  const overrideFor = (canonical: string, adapter: string) => {
    if (!overrides.data?.ok) return undefined;
    return overrides.data.data.find(
      (o) => o.canonicalVoice === canonical && o.tierProvider === adapter,
    );
  };

  const startEdit = (target: EditTarget, current: VoiceOverrideRow | undefined) => {
    setEditing(target);
    setDraft(current?.adapterVoice ?? "");
    setSaveError(null);
  };

  const cancelEdit = () => {
    setEditing(null);
    setDraft("");
    setSaveError(null);
  };

  const commitEdit = () => {
    if (!editing || saveOverride.isPending) return;
    setSaveError(null);
    // An empty value deletes the override server-side; the adapter falls back
    // to its own default voice (see the Settings proto contract).
    saveOverride.mutate({ ...editing, adapterVoice: draft.trim() });
  };

  return (
    <div className="flex max-w-6xl flex-col gap-4 md:gap-6">
      <PageHeader title={t(strings.voices.title)} description={t(strings.voices.description)} />

      <Panel title={t(strings.voices.matrixTitle)} bodyless>
        {overrides.isLoading ? (
          <div className="p-4"><LoadingRows rows={5} label={t(strings.common.loading)} /></div>
        ) : overrides.data && !overrides.data.ok ? (
          <div className="p-4">
            <ApiErrorState error={overrides.data.error} onRetry={() => void overrides.refetch()} />
          </div>
        ) : (
          <Table>
            <THead>
              <TR>
                <TH>{t(strings.voices.canonicalHeader)}</TH>
                {TTS_ADAPTERS.map((a) => (
                  <TH key={a}>{a}</TH>
                ))}
              </TR>
            </THead>
            <TBody>
              {CANONICAL_VOICES.map((c) => (
                <TR key={c}>
                  <TD className="font-mono text-xs">{c}</TD>
                  {TTS_ADAPTERS.map((a) => {
                    const o = overrideFor(c, a);
                    const isEditing =
                      editing !== null &&
                      editing.canonicalVoice === c &&
                      editing.tierProvider === a;
                    const isSaving = isEditing && saveOverride.isPending;
                    const labelVars = { voice: c, adapter: a };

                    return (
                      <TD key={a} className="font-mono text-xs" aria-busy={isSaving}>
                        {isEditing ? (
                          <form
                            className="flex flex-col gap-1"
                            onSubmit={(e) => {
                              e.preventDefault();
                              commitEdit();
                            }}
                          >
                            <Input
                              ref={editorRef}
                              value={draft}
                              disabled={isSaving}
                              aria-label={t(strings.voices.editorAria, labelVars)}
                              placeholder={t(strings.voices.placeholder)}
                              onChange={(e) => setDraft(e.target.value)}
                              onKeyDown={(e) => {
                                if (e.key === "Escape") {
                                  e.preventDefault();
                                  cancelEdit();
                                }
                              }}
                              className="h-8 min-w-24"
                            />
                            <div className="flex flex-wrap gap-1">
                              <Button
                                type="submit"
                                size="sm"
                                disabled={isSaving}
                                aria-label={t(strings.voices.saveAria, labelVars)}
                              >
                                {isSaving ? (
                                  <Loader2 className="h-3 w-3 animate-spin" aria-hidden="true" />
                                ) : null}
                                {isSaving
                                  ? t(strings.voices.saving)
                                  : t(strings.voices.saveButton)}
                              </Button>
                              <Button
                                type="button"
                                size="sm"
                                variant="ghost"
                                disabled={isSaving}
                                aria-label={t(strings.voices.cancelAria, labelVars)}
                                onClick={cancelEdit}
                              >
                                {t(strings.voices.cancelButton)}
                              </Button>
                            </div>
                            {saveError ? (
                              <p role="alert" className="text-xs text-app-danger">
                                {saveError}
                              </p>
                            ) : null}
                          </form>
                        ) : (
                          <button
                            type="button"
                            disabled={saveOverride.isPending}
                            aria-label={t(strings.voices.editAria, labelVars)}
                            title={t(strings.voices.editButton)}
                            onClick={() => startEdit({ canonicalVoice: c, tierProvider: a }, o)}
                            className="flex w-full items-center justify-between gap-1 rounded-control px-1 py-0.5 text-left transition-colors hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-focus/60 disabled:cursor-not-allowed disabled:opacity-60"
                          >
                            <span className={o ? "" : "text-app-muted-foreground"}>
                              {o ? o.adapterVoice : t(strings.common.default)}
                            </span>
                            <Pencil
                              className="h-3 w-3 shrink-0 text-app-muted-foreground"
                              aria-hidden="true"
                            />
                          </button>
                        )}
                      </TD>
                    );
                  })}
                </TR>
              ))}
            </TBody>
          </Table>
        )}
      </Panel>

      <Panel
        title={t(strings.voices.localHealthTitle)}
        description={t(strings.voices.localHealthDescription)}
      >
        <ul className="flex flex-col gap-2 text-sm">
          {probes.length === 0 ? (
            <li className="text-xs text-app-muted-foreground">{t(strings.common.loading)}</li>
          ) : (
            probes.map((r) => (
              <li
                key={r.id}
                className="flex items-center justify-between rounded-control border border-app-border bg-app-surface-muted px-3 py-2"
              >
                <span className="flex items-center gap-2">
                  <span className="font-mono text-xs">{r.id}</span>
                  <span className="text-xs text-app-muted-foreground">{r.role}</span>
                </span>
                <StatusDot tone={r.tone} label={r.label} />
              </li>
            ))
          )}
        </ul>
        <p className="mt-3 text-xs text-app-muted-foreground">{t(strings.voices.localHealthFollowUp)}</p>
      </Panel>
    </div>
  );
}
