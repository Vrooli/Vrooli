import { useQuery } from "@tanstack/react-query";
import { Panel } from "../../components/ui/panel";
import { Badge } from "../../components/ui/badge";
import { StatusDot } from "../../components/ui/status-dot";
import { Table, TBody, TD, TH, THead, TR } from "../../components/ui/table";
import { PageHeader } from "../../components/composites/PageHeader";
import { ApiErrorState } from "../../components/composites/ApiErrorState";
import { LoadingRows } from "../../components/composites/LoadingRows";
import { getProviderConfig, listByokCredentials } from "../../services/settings";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

const CAPABILITIES = ["STT", "TTS", "Summarize"] as const;

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
  const provider = useQuery({ queryKey: ["settings", "provider"], queryFn: getProviderConfig });
  const creds = useQuery({ queryKey: ["settings", "byok"], queryFn: listByokCredentials });

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
        ) : null}
      </Panel>

      <Panel title={t(strings.config.byokCredsTitle)} description={t(strings.config.byokCredsDescription)} bodyless>
        {creds.isLoading ? (
          <div className="p-4"><LoadingRows rows={3} label={t(strings.common.loading)} /></div>
        ) : creds.data && !creds.data.ok ? (
          <div className="p-4">
            <ApiErrorState error={creds.data.error} onRetry={() => void creds.refetch()} />
          </div>
        ) : creds.data?.ok && creds.data.data.length === 0 ? (
          <div className="p-4 text-sm text-app-muted-foreground">{t(strings.config.byokEmpty)}</div>
        ) : creds.data?.ok ? (
          <Table>
            <THead>
              <TR>
                <TH>{t(strings.config.providerColHeader)}</TH>
                <TH>{t(strings.config.capabilityHeader)}</TH>
                <TH>{t(strings.config.fingerprintHeader)}</TH>
                <TH>{t(strings.config.createdHeader)}</TH>
              </TR>
            </THead>
            <TBody>
              {creds.data.data.map((c) => (
                <TR key={`${c.providerId}-${c.capability}`}>
                  <TD className="font-mono text-xs">{c.providerId}</TD>
                  <TD><Badge variant="info">{c.capability}</Badge></TD>
                  <TD className="font-mono text-xs text-app-muted-foreground">{c.fingerprint}</TD>
                  <TD className="text-xs text-app-muted-foreground">{c.createdAt || t(strings.common.dash)}</TD>
                </TR>
              ))}
            </TBody>
          </Table>
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

function TierBadge({ enabled }: { enabled: boolean }) {
  const { t } = useTranslation();
  return enabled ? (
    <StatusDot tone="success" label={t(strings.status.enabled)} />
  ) : (
    <StatusDot tone="neutral" label={t(strings.status.off)} />
  );
}
