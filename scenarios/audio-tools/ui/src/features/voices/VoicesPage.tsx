import { useQuery } from "@tanstack/react-query";
import { Panel } from "../../components/ui/panel";
import { Table, TBody, TD, TH, THead, TR } from "../../components/ui/table";
import { StatusDot } from "../../components/ui/status-dot";
import { PageHeader } from "../../components/composites/PageHeader";
import { ApiErrorState } from "../../components/composites/ApiErrorState";
import { LoadingRows } from "../../components/composites/LoadingRows";
import { getVoiceOverrides } from "../../services/settings";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

const CANONICAL_VOICES = [
  "voice.feminine.warm",
  "voice.feminine.neutral",
  "voice.masculine.warm",
  "voice.masculine.neutral",
  "voice.neutral.default",
];

const ADAPTERS = ["local:kokoro", "byok:openai-tts", "byok:elevenlabs"];

const LOCAL_RESOURCES = [
  { id: "whisper", role: "STT" },
  { id: "kokoro", role: "TTS" },
  { id: "ollama", role: "Summarize" },
];

export function VoicesPage() {
  const { t } = useTranslation();
  const overrides = useQuery({ queryKey: ["settings", "voices"], queryFn: getVoiceOverrides });

  const overrideFor = (canonical: string, adapter: string) => {
    if (!overrides.data?.ok) return undefined;
    return overrides.data.data.find(
      (o) => o.canonicalVoice === canonical && o.tierProvider === adapter,
    );
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
                {ADAPTERS.map((a) => (
                  <TH key={a}>{a}</TH>
                ))}
              </TR>
            </THead>
            <TBody>
              {CANONICAL_VOICES.map((c) => (
                <TR key={c}>
                  <TD className="font-mono text-xs">{c}</TD>
                  {ADAPTERS.map((a) => {
                    const o = overrideFor(c, a);
                    return (
                      <TD key={a} className="font-mono text-xs">
                        {o ? o.adapterVoice : (
                          <span className="text-app-muted-foreground">{t(strings.common.default)}</span>
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
          {LOCAL_RESOURCES.map((r) => (
            <li
              key={r.id}
              className="flex items-center justify-between rounded-control border border-app-border bg-app-surface-muted px-3 py-2"
            >
              <span className="flex items-center gap-2">
                <span className="font-mono text-xs">{r.id}</span>
                <span className="text-xs text-app-muted-foreground">{r.role}</span>
              </span>
              <StatusDot tone="neutral" label={t(strings.status.probePending)} />
            </li>
          ))}
        </ul>
        <p className="mt-3 text-xs text-app-muted-foreground">{t(strings.voices.localHealthFollowUp)}</p>
      </Panel>
    </div>
  );
}
