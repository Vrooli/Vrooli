import { useEffect, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { readInstallPosture, writeUpdateControl, type UpdateControlMode } from "../../api/installPosture";
import { Button } from "@vrooli/react-component-library/Button/2";
import { i18n } from "../../i18n";

export function InstallPosture() {
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: ["install-posture"], queryFn: readInstallPosture });
  const [draft, setDraft] = useState<UpdateControlMode | "">("");
  const [saving, setSaving] = useState(false);
  const [writeError, setWriteError] = useState("");
  useEffect(() => setDraft(query.data?.updateControl ?? ""), [query.data?.updateControl]);

  const save = async () => {
    if (!query.data || !draft || draft === query.data.updateControl) return;
    setSaving(true);
    setWriteError("");
    try {
      await writeUpdateControl(draft, query.data.updateControlRevision, crypto.randomUUID());
      await queryClient.invalidateQueries({ queryKey: ["install-posture"] });
    } catch (error) {
      setWriteError(error instanceof Error ? error.message : i18n.t("onboarding.dashboard.postureWriteError"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <section className="mt-8 rounded-xl border border-border bg-surface p-5" aria-labelledby="install-posture-heading" data-testid="install-posture">
      <p className="text-xs font-medium uppercase tracking-wide text-muted">{i18n.t("onboarding.dashboard.installPostureEyebrow")}</p>
      <h2 id="install-posture-heading" className="mt-1 text-lg font-semibold">{i18n.t("onboarding.dashboard.installPostureHeading")}</h2>
      <p className="mt-1 text-sm text-muted">{i18n.t("onboarding.dashboard.installPostureIntro")}</p>
      {query.isLoading && <p className="mt-4 text-sm text-muted" role="status">{i18n.t("onboarding.dashboard.installPostureLoading")}</p>}
      {query.error && <p className="mt-4 text-sm text-danger" role="alert">{i18n.t("onboarding.dashboard.installPostureReadError")}</p>}
      {query.data && <div className="mt-4 grid gap-4 sm:grid-cols-3">
        <div className="grid content-start gap-2 text-sm">
          <span className="font-medium">{i18n.t("onboarding.dashboard.trustPosture")}</span>
          <span data-testid="install-trust-posture">{query.data.trustPosture}</span>
          <span className="text-xs text-muted">{i18n.t("onboarding.dashboard.trustPostureOwnerOnly")}</span>
        </div>
        <label className="grid gap-2 text-sm">
          <span className="font-medium">{i18n.t("onboarding.dashboard.updateControl")}</span>
          <select className="rounded-md border border-border bg-surface px-3 py-2" value={draft} onChange={(event) => setDraft(event.target.value as UpdateControlMode)} aria-label={i18n.t("onboarding.dashboard.updateControl")}>
            <option value="observe">{i18n.t("onboarding.dashboard.updateControlObserve")}</option>
            <option value="guard">{i18n.t("onboarding.dashboard.updateControlGuard")}</option>
          </select>
        </label>
        <div className="grid content-start gap-2 text-sm">
          <span className="font-medium">{i18n.t("onboarding.dashboard.sessionMode")}</span>
          <span data-testid="install-session-mode">{query.data.sessionMode ?? i18n.t("onboarding.dashboard.sessionModeUnavailable")}</span>
          <span className="text-xs text-muted">{i18n.t("onboarding.dashboard.sessionModeOwnerOnly")}</span>
        </div>
      </div>}
      {writeError && <p className="mt-3 text-sm text-danger" role="alert">{writeError}</p>}
      {query.data && <Button variant="secondary" size="sm" className="mt-4" onClick={() => void save()} disabled={saving || !draft || draft === query.data.updateControl}>
        {saving ? i18n.t("onboarding.dashboard.updateControlSaving") : i18n.t("onboarding.dashboard.updateControlSave")}
      </Button>}
    </section>
  );
}
