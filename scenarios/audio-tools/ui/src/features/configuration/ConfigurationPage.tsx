import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Panel } from "../../components/ui/panel";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import { Input } from "@vrooli/react-component-library/Input/1.3.6";
import { Table, TBody, TD, TH, THead, TR } from "../../components/ui/table";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { ApiErrorState } from "../../components/composites/ApiErrorState";
import { CopyableCommand } from "../../components/composites/CopyableCommand";
import { LoadingRows } from "../../components/composites/LoadingRows";
import {
  deleteByokCredential,
  getProviderConfig,
  listByokCredentials,
  updateProviderConfig,
  upsertByokCredential,
} from "../../services/settings";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { PROVIDERS, type ProviderMeta } from "../providers/providerCatalog.generated";
import { TierBadge } from "./components/TierBadge";

const CAPABILITIES = ["STT", "TTS", "Summarize", "Duplex"] as const;
export const DUPLEX_CREDENTIAL_PROVISION_COMMAND =
  "vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key";
const DUPLEX_CREDENTIAL_OBTAIN_URL = "https://platform.openai.com/api-keys";

/**
 * This legacy BYOK form stores provider keys in Audio Tools' settings store.
 * Full-duplex credentials use Vrooli's canonical credential authority instead.
 */
const CREDENTIAL_PROVIDER_KINDS: ReadonlySet<ProviderMeta["kind"]> = new Set(["byok_api", "gateway"]);

const CAPABILITY_LABEL_KEYS = {
  stt: strings.config.capabilityStt,
  tts: strings.config.capabilityTts,
  summarize: strings.config.capabilitySummarize,
  duplex: strings.config.capabilityDuplex,
} as const;

type ByokCapability = keyof typeof CAPABILITY_LABEL_KEYS;

const credentialProvidersFor = (capability: string): ProviderMeta[] =>
  Object.values(PROVIDERS).filter(
    (provider) =>
      CREDENTIAL_PROVIDER_KINDS.has(provider.kind) &&
      (provider.capabilities as readonly string[]).includes(capability),
  );

const ENV_VARS = [
  "AUDIO_AI_ENABLE_BYOK",
  "AUDIO_AI_ENABLE_VROOLI",
  "AUDIO_AI_ENABLE_LOCAL",
  "AUDIO_WHISPER_URL",
  "AUDIO_KOKORO_URL",
  "AUDIO_OLLAMA_URL",
  "AUDIO_LPBS_BASE_URL",
  "AUDIO_AVAIL_TTL_BYOK",
  "AUDIO_AVAIL_TTL_VROOLI",
];

export function ConfigurationPage() {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const provider = useQuery({ queryKey: ["settings", "provider"], queryFn: getProviderConfig });
  const creds = useQuery({ queryKey: ["settings", "byok"], queryFn: listByokCredentials });

  const updateMut = useMutation({
    mutationFn: updateProviderConfig,
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["settings", "provider"] }),
  });
  const upsertMut = useMutation({
    mutationFn: ({ providerId, capability, apiKey }: { providerId: string; capability: string; apiKey: string }) =>
      upsertByokCredential(providerId, capability, apiKey),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["settings", "byok"] }),
  });
  const deleteMut = useMutation({
    mutationFn: ({ providerId, capability }: { providerId: string; capability: string }) => deleteByokCredential(providerId, capability),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["settings", "byok"] }),
  });

  const [byokForm, setByokForm] = useState<{ providerId: string; capability: ByokCapability; apiKey: string }>({
    providerId: "",
    capability: "tts",
    apiKey: "",
  });

  const credentialProviders = credentialProvidersFor(byokForm.capability);
  const selectedProvider = credentialProviders.find((provider) => provider.id === byokForm.providerId);
  const capabilityLabel = t(CAPABILITY_LABEL_KEYS[byokForm.capability]);
  const canSubmitByok = Boolean(selectedProvider && byokForm.apiKey);

  return (
    <div className="flex max-w-6xl flex-col gap-4 md:gap-6">
      <PageHeader title={t(strings.config.title)} description={t(strings.config.description)} />

      <Panel
        title={t(strings.config.providerRoutingTitle)}
        description={t(strings.config.providerRoutingDescription)}
        bodyless
      >
        {provider.isLoading ? (
          <div className="p-4"><LoadingRows rows={3} label={t(strings.common.loading)} /></div>
        ) : provider.data && !provider.data.ok ? (
          <div className="p-4">
            <ApiErrorState error={provider.data.error} onRetry={() => void provider.refetch()} />
          </div>
        ) : provider.data?.ok ? (
          <div>
            <Table>
              <THead>
                <TR>
                  <TH>{t(strings.config.capabilityHeader)}</TH>
                  <TH>{t(strings.config.byokHeader)}</TH>
                  <TH>{t(strings.config.vrooliHeader)}</TH>
                  <TH>{t(strings.config.localHeader)}</TH>
                </TR>
              </THead>
              <TBody>
                {CAPABILITIES.map((cap) => (
                  <TR key={cap}>
                    <TD className="font-medium">{cap}</TD>
                    <TD><TierBadge enabled={provider.data.ok ? provider.data.data.byokEnabled : false} /></TD>
                    <TD><TierBadge enabled={provider.data.ok ? provider.data.data.vrooliEnabled : false} /></TD>
                    <TD><TierBadge enabled={provider.data.ok ? provider.data.data.localEnabled : false} /></TD>
                  </TR>
                ))}
              </TBody>
            </Table>
            <div className="flex flex-wrap gap-2 px-4 py-3">
              <Button size="sm" variant="outline" disabled={updateMut.isPending}
                onClick={() => provider.data.ok && updateMut.mutate({ byokEnabled: !provider.data.data.byokEnabled })}>
                {t(strings.config.toggleByok)}
              </Button>
              <Button size="sm" variant="outline" disabled={updateMut.isPending}
                onClick={() => provider.data.ok && updateMut.mutate({ vrooliEnabled: !provider.data.data.vrooliEnabled })}>
                {t(strings.config.toggleVrooli)}
              </Button>
              <Button size="sm" variant="outline" disabled={updateMut.isPending}
                onClick={() => provider.data.ok && updateMut.mutate({ localEnabled: !provider.data.data.localEnabled })}>
                {t(strings.config.toggleLocal)}
              </Button>
            </div>
          </div>
        ) : null}
      </Panel>

      <Panel title={t(strings.config.byokCredsTitle)} description={t(strings.config.byokCredsDescription)} bodyless>
        {creds.isLoading ? (
          <div className="p-4"><LoadingRows rows={3} label={t(strings.common.loading)} /></div>
        ) : creds.data && !creds.data.ok ? (
          <div className="p-4">
            <ApiErrorState error={creds.data.error} onRetry={() => void creds.refetch()} />
          </div>
        ) : creds.data?.ok ? (
          <div>
            {creds.data.data.length === 0 ? (
              <div className="p-4 text-sm text-app-muted-foreground">{t(strings.config.byokEmpty)}</div>
            ) : (
              <Table>
                <THead>
                  <TR>
                    <TH>{t(strings.config.providerColHeader)}</TH>
                    <TH>{t(strings.config.capabilityHeader)}</TH>
                    <TH>{t(strings.config.fingerprintHeader)}</TH>
                    <TH>{t(strings.config.createdHeader)}</TH>
                    <TH></TH>
                  </TR>
                </THead>
                <TBody>
                  {creds.data.data.map((c) => (
                    <TR key={`${c.providerId}-${c.capability}`}>
                      <TD className="font-mono text-xs">{c.providerId}</TD>
                      <TD><Badge variant="info">{c.capability}</Badge></TD>
                      <TD className="font-mono text-xs text-app-muted-foreground">{c.fingerprint}</TD>
                      <TD className="text-xs text-app-muted-foreground">{c.createdAt || t(strings.common.dash)}</TD>
                      <TD>
                        <Button size="sm" variant="ghost" onClick={() => deleteMut.mutate({ providerId: c.providerId, capability: c.capability })}>
                          {t(strings.config.deleteAction)}
                        </Button>
                      </TD>
                    </TR>
                  ))}
                </TBody>
              </Table>
            )}
            <form
              className="flex flex-wrap items-end gap-2 px-4 py-3"
              onSubmit={(e) => {
                e.preventDefault();
                if (!selectedProvider || !byokForm.apiKey) return;
                upsertMut.mutate(byokForm);
                setByokForm({ ...byokForm, apiKey: "" });
              }}
            >
              <label htmlFor="byok-capability" className="flex flex-col gap-1 text-xs">
                {t(strings.config.capabilityFieldLabel)}
                <select
                  id="byok-capability"
                  className="rounded-control border border-app-border bg-app-surface px-2 py-1"
                  value={byokForm.capability}
                  onChange={(e) => {
                    const capability = e.currentTarget.value as ByokCapability;
                    setByokForm((previous) => ({
                      ...previous,
                      capability,
                      providerId: credentialProvidersFor(capability).some(
                        (provider) => provider.id === previous.providerId,
                      ) ? previous.providerId : "",
                    }));
                  }}
                >
                  <option value="stt">{t(strings.config.capabilityStt)}</option>
                  <option value="tts">{t(strings.config.capabilityTts)}</option>
                  <option value="summarize">{t(strings.config.capabilitySummarize)}</option>
                  <option value="duplex">{t(strings.config.capabilityDuplex)}</option>
                </select>
              </label>
              {byokForm.capability === "duplex" ? (
                <div className="flex min-w-[280px] flex-1 flex-col gap-2 text-sm" role="status">
                  <p className="text-app-muted-foreground">{t(strings.config.duplexCredentialInstructions)}</p>
                  <Button asChild size="sm" variant="link" className="w-fit px-0">
                    <a href={DUPLEX_CREDENTIAL_OBTAIN_URL} target="_blank" rel="noopener noreferrer">
                      {t(strings.config.duplexCredentialOpenPlatform)}
                    </a>
                  </Button>
                  <CopyableCommand
                    command={DUPLEX_CREDENTIAL_PROVISION_COMMAND}
                    copyLabel={t(strings.streamConfigAdmin.copyCommand)}
                    copiedLabel={t(strings.streamConfigAdmin.copied)}
                  />
                  <p className="text-xs text-app-muted-foreground">{t(strings.config.duplexCredentialStatus)}</p>
                </div>
              ) : (
                <>
                  <div className="flex flex-col gap-1 text-xs">
                    <label htmlFor="byok-provider">{t(strings.config.providerFieldLabel)}</label>
                    {credentialProviders.length > 0 ? (
                      <>
                        <select
                          id="byok-provider"
                          className="rounded-control border border-app-border bg-app-surface px-2 py-1"
                          value={byokForm.providerId}
                          aria-describedby="byok-provider-hint"
                          onChange={(e) => setByokForm({ ...byokForm, providerId: e.currentTarget.value })}
                        >
                          <option value="">{t(strings.config.providerSelectPlaceholder)}</option>
                          {credentialProviders.map((provider) => (
                            <option key={provider.id} value={provider.id}>
                              {`${provider.name} (${provider.id})`}
                            </option>
                          ))}
                        </select>
                        <p id="byok-provider-hint" className="text-[11px] text-app-muted-foreground">
                          {selectedProvider
                            ? t(strings.config.providerSelectedHint, {
                                provider: selectedProvider.name,
                                capability: capabilityLabel,
                              })
                            : t(strings.config.providerSelectHint, { capability: capabilityLabel })}
                        </p>
                        {selectedProvider && selectedProvider.status !== "active" ? (
                          <p role="status" className="max-w-sm text-[11px] text-app-warning">
                            {selectedProvider.description}
                          </p>
                        ) : null}
                      </>
                    ) : (
                      <p id="byok-provider-empty" role="status" className="max-w-xs text-[11px] text-app-warning">
                        {t(strings.config.providerNoneForCapability, { capability: capabilityLabel })}
                      </p>
                    )}
                  </div>
                  <label htmlFor="byok-apikey" className="flex min-w-[180px] flex-1 flex-col gap-1 text-xs">
                    {t(strings.config.apiKeyFieldLabel)}
                    <Input
                      id="byok-apikey"
                      type="password"
                      value={byokForm.apiKey}
                      onChange={(e) => setByokForm({ ...byokForm, apiKey: e.currentTarget.value })}
                      placeholder={t(strings.config.apiKeyPlaceholder)}
                    />
                  </label>
                  <Button type="submit" disabled={upsertMut.isPending || !canSubmitByok}>
                    {t(strings.config.addCredential)}
                  </Button>
                </>
              )}
            </form>
          </div>
        ) : null}
      </Panel>

      <Panel title={t(strings.config.envTitle)} description={t(strings.config.envDescription)}>
        <ul className="grid gap-2 text-sm font-mono md:grid-cols-2">
          {ENV_VARS.map((v) => (
            <li key={v} className="rounded-control bg-app-surface-muted px-2 py-1 text-app-muted-foreground">
              {v}
            </li>
          ))}
        </ul>
        <p className="mt-3 text-xs text-app-muted-foreground">{t(strings.config.envFollowUp)}</p>
      </Panel>
    </div>
  );
}
