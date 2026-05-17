import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Panel } from "../../components/ui/panel";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { StatusDot } from "../../components/ui/status-dot";
import { Table, TBody, TD, TH, THead, TR } from "../../components/ui/table";
import { PageHeader } from "../../components/composites/PageHeader";
import { ApiErrorState } from "../../components/composites/ApiErrorState";
import { LoadingRows } from "../../components/composites/LoadingRows";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import {
  deleteSpeakerProfile,
  getSpeakerStatus,
  unbindSpeakerProfile,
  updateSpeakerConfig,
  type SpeakerModeLabel,
  type RejectBehaviorLabel,
} from "../../services/speakerAdmin";

const capabilityTone = (capability: string): "success" | "warning" | "danger" | "neutral" => {
  switch (capability) {
    case "available":
      return "success";
    case "degraded":
      return "warning";
    case "unavailable":
      return "danger";
    default:
      return "neutral";
  }
};

export function SpeakerVerificationPage() {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const status = useQuery({ queryKey: ["speaker", "status"], queryFn: getSpeakerStatus });

  const saveMut = useMutation({
    mutationFn: updateSpeakerConfig,
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["speaker", "status"] }),
  });
  const unbindMut = useMutation({
    mutationFn: unbindSpeakerProfile,
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["speaker", "status"] }),
  });
  const deleteMut = useMutation({
    mutationFn: deleteSpeakerProfile,
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["speaker", "status"] }),
  });

  if (status.isPending) {
    return (
      <div className="flex flex-col gap-4">
        <PageHeader
          title={t(strings.speakerAdmin.pageTitle)}
          description={t(strings.speakerAdmin.pageDescription)}
        />
        <LoadingRows />
      </div>
    );
  }

  if (status.isError) {
    return (
      <div className="flex flex-col gap-4">
        <PageHeader
          title={t(strings.speakerAdmin.pageTitle)}
          description={t(strings.speakerAdmin.pageDescription)}
        />
        <ApiErrorState error={status.error} onRetry={() => status.refetch()} />
      </div>
    );
  }

  const st = status.data;
  const cfg = st.config;

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={t(strings.speakerAdmin.pageTitle)}
        description={t(strings.speakerAdmin.pageDescription)}
      />

      <Panel title="Status">
        <div className="flex items-center gap-2 px-4 pt-3 text-xs">
          <StatusDot tone={capabilityTone(st.capability)} label={st.capabilityLabel || st.capability} />
          <span className="text-app-muted-foreground">
            {st.profileCount} {st.profileCount === 1 ? "profile" : "profiles"}
          </span>
        </div>
        <dl className="grid grid-cols-2 gap-2 px-4 py-3 text-xs">
          <dt className="text-app-muted-foreground">Resource ready</dt>
          <dd>{String(st.resourceReady)}</dd>
          <dt className="text-app-muted-foreground">Profile configured</dt>
          <dd>{String(st.profileConfigured)}</dd>
          <dt className="text-app-muted-foreground">Profile exists</dt>
          <dd>{String(st.profileExists)}</dd>
        </dl>
      </Panel>

      <Panel title={t(strings.speakerAdmin.configEnabled)}>
        <form
          className="flex flex-col gap-3"
          onSubmit={(e) => {
            e.preventDefault();
            const fd = new FormData(e.currentTarget);
            const enabled = fd.get("enabled") === "on";
            const threshold = parseFloat(String(fd.get("threshold") ?? "0"));
            const mode = String(fd.get("mode") ?? "filter") as SpeakerModeLabel;
            const rejectBehavior = String(fd.get("rejectBehavior") ?? "drop") as RejectBehaviorLabel;
            saveMut.mutate({ enabled, threshold, mode, rejectBehavior });
          }}
        >
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" name="enabled" defaultChecked={cfg.enabled} />
            {t(strings.speakerAdmin.configEnabled)}
          </label>
          <label className="flex flex-col gap-1 text-sm">
            {t(strings.speakerAdmin.configThreshold)}
            <Input
              type="number"
              step="0.01"
              min={0}
              max={1}
              name="threshold"
              defaultValue={cfg.threshold || 0}
            />
            <span className="text-xs text-app-muted-foreground">{t(strings.speakerAdmin.thresholdHelp)}</span>
          </label>
          <label className="flex flex-col gap-1 text-sm">
            {t(strings.speakerAdmin.configMode)}
            <select
              name="mode"
              defaultValue={cfg.mode}
              className="rounded-control border border-app-border bg-app-surface px-2 py-1"
            >
              <option value="off">{t(strings.speakerAdmin.modeOff)}</option>
              <option value="filter">{t(strings.speakerAdmin.modeFilter)}</option>
              <option value="advisory">{t(strings.speakerAdmin.modeAdvisory)}</option>
            </select>
          </label>
          <label className="flex flex-col gap-1 text-sm">
            {t(strings.speakerAdmin.configRejectBehavior)}
            <select
              name="rejectBehavior"
              defaultValue={cfg.rejectBehavior}
              className="rounded-control border border-app-border bg-app-surface px-2 py-1"
            >
              <option value="drop">{t(strings.speakerAdmin.rejectDrop)}</option>
              <option value="show-muted">{t(strings.speakerAdmin.rejectShowMuted)}</option>
            </select>
          </label>
          <div>
            <Button type="submit" disabled={saveMut.isPending}>
              {t(strings.speakerAdmin.saveChanges)}
            </Button>
          </div>
        </form>
      </Panel>

      <Panel title={t(strings.speakerAdmin.profilesTitle)}>
        {st.profiles.length === 0 ? (
          <p className="px-4 py-3 text-sm text-app-muted-foreground">{t(strings.speakerAdmin.profilesEmpty)}</p>
        ) : (
          <Table>
            <THead>
              <TR>
                <TH>ID</TH>
                <TH>Name</TH>
                <TH>Model</TH>
                <TH>Sample rate</TH>
                <TH aria-label="actions" />
              </TR>
            </THead>
            <TBody>
              {st.profiles.map((p) => (
                <TR key={p.id}>
                  <TD className="font-mono text-xs">{p.id}</TD>
                  <TD>{p.displayName}</TD>
                  <TD>{p.modelName}</TD>
                  <TD>{p.sampleRate}</TD>
                  <TD className="flex gap-2">
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => unbindMut.mutate(p.id)}
                      disabled={unbindMut.isPending}
                    >
                      {t(strings.speakerAdmin.unbindButton)}
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => deleteMut.mutate(p.id)}
                      disabled={deleteMut.isPending}
                    >
                      {t(strings.speakerAdmin.deleteButton)}
                    </Button>
                  </TD>
                </TR>
              ))}
            </TBody>
          </Table>
        )}
      </Panel>
    </div>
  );
}
