import { useCallback, useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Check, HardDrive, ShieldCheck, Undo2 } from "lucide-react";
import { fetchHostFacts, type HostFacts } from "../../api/host";
import { StatCard } from "@vrooli/react-component-library/StatCard/1";
import { StepPlan } from "./StepPlan";
import { i18n } from "../../i18n";
import { evaluateProfile, fetchProfiles } from "../../api/profiles";
import { useOnboardingTargetCapability } from "../../api/useOnboardingTargetCapability";
import type { EvaluateProfileResponse, ProfileQuestion } from "@vrooli/proto-types/vrooli-onboarding/v1/profiles/profiles_pb";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Checkbox } from "@vrooli/react-component-library/Checkbox/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { Select } from "@vrooli/react-component-library/Select/1";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import type { WizardProfileSession, WizardProfileSessionSaveRequest } from "../../api/session";

type WelcomeProps = {
  target?: string;
  onAccept?: (profile?: string, scenarios?: string[]) => Promise<void>;
  onAdjust?: () => void;
  profileSession?: WizardProfileSession | null;
  profileSessionBaseRevision?: string;
  onProfileSessionChange?: (draft: Omit<WizardProfileSessionSaveRequest, "expectedRevision">) => void;
  profileSessionError?: string | null;
  profileSessionSaveState?: "idle" | "saving" | "saved" | "failed" | "conflict";
  onRetryProfileSessionSave?: () => void;
};

type ProfileDraftState = {
  profileId: string;
  answers: Record<string, unknown>;
  manualDecisions: Record<string, boolean>;
  evaluation: EvaluateProfileResponse | null;
  error: string | null;
  evaluating: boolean;
  dirty: boolean;
  appliedSessionMarker: string | null;
};

const emptyProfileDraft: ProfileDraftState = {
  profileId: "",
  answers: {},
  manualDecisions: {},
  evaluation: null,
  error: null,
  evaluating: false,
  dirty: false,
  appliedSessionMarker: null,
};

export function StepWelcome({ target = "local", onAccept, onAdjust, profileSession, profileSessionBaseRevision = "", onProfileSessionChange, profileSessionError, profileSessionSaveState = "idle", onRetryProfileSessionSave }: WelcomeProps = {}) {
  const [hostFacts, setHostFacts] = useState<HostFacts | null>(null);

  useEffect(() => {
    let active = true;
    fetchHostFacts(target).then((facts) => {
      if (active) setHostFacts(facts);
    }).catch(() => {
      if (active) setHostFacts({ available: false });
    });
    return () => { active = false; };
  }, [target]);

  return (
    <div className="welcome-screen" data-testid="step-welcome">
      <p className="welcome-screen__eyebrow"><Check aria-hidden="true" /> {i18n.t("onboarding.welcome.eyebrow")}</p>
      <h1>{i18n.t("onboarding.welcome.heading")}</h1>
      <p className="welcome-screen__lede">
        {i18n.t("onboarding.welcome.lede")}
      </p>
      <div className="commitment-list" aria-label={i18n.t("onboarding.welcome.commitments")}>
        <Commitment icon={<HardDrive aria-hidden="true" />} title={i18n.t("onboarding.welcome.servicesTitle")} description={i18n.t("onboarding.welcome.servicesDescription")} />
        <Commitment icon={<ShieldCheck aria-hidden="true" />} title={i18n.t("onboarding.welcome.hostTitle")} description={i18n.t("onboarding.welcome.hostDescription")} />
        <Commitment icon={<Undo2 aria-hidden="true" />} title={i18n.t("onboarding.welcome.reversibleTitle")} description={i18n.t("onboarding.welcome.reversibleDescription")} />
      </div>
      {hostFacts?.available && <section className="host-facts" data-testid="host-facts" aria-label={i18n.t("onboarding.welcome.hostFacts")}>
        <div className="section-divider"><span>{i18n.t("onboarding.welcome.detected")}</span></div>
        <div className="host-facts__grid">
        <StatCard label={i18n.t("onboarding.welcome.memory")} value={formatBytes(hostFacts.memory_total_bytes)} />
        <StatCard label={i18n.t("onboarding.welcome.freeDisk")} value={formatBytes(hostFacts.disk_free_bytes)} />
        <StatCard label={i18n.t("onboarding.welcome.gpu")} value={hostFacts.gpus?.[0] ?? i18n.t("onboarding.welcome.noneDetected")} />
        <StatCard label={i18n.t("onboarding.welcome.platform")} value={hostFacts.platform ?? i18n.t("onboarding.welcome.unknown")} />
        </div>
      </section>}
      {onAccept && onAdjust && <PurposeProfile target={target} onAccept={onAccept} onAdjust={onAdjust} profileSession={profileSession} profileSessionBaseRevision={profileSessionBaseRevision} onProfileSessionChange={onProfileSessionChange} profileSessionError={profileSessionError} profileSessionSaveState={profileSessionSaveState} onRetryProfileSessionSave={onRetryProfileSessionSave} />}
    </div>
  );
}

function PurposeProfile({ target = "local", onAccept, onAdjust, profileSession, profileSessionBaseRevision = "", onProfileSessionChange, profileSessionError, profileSessionSaveState = "idle", onRetryProfileSessionSave }: { target?: string; onAccept: WelcomeProps["onAccept"]; onAdjust: () => void; profileSession?: WizardProfileSession | null; profileSessionBaseRevision?: string; onProfileSessionChange?: WelcomeProps["onProfileSessionChange"]; profileSessionError?: string | null; profileSessionSaveState?: WelcomeProps["profileSessionSaveState"]; onRetryProfileSessionSave?: () => void }) {
  const [draftsByTarget, setDraftsByTarget] = useState<Map<string, ProfileDraftState>>(() => new Map());
  const updateTargetDraft = useCallback((targetId: string, update: (current: ProfileDraftState) => ProfileDraftState) => {
    setDraftsByTarget((current) => {
      const next = new Map(current);
      next.set(targetId, update(current.get(targetId) ?? emptyProfileDraft));
      return next;
    });
  }, []);
  const draft = draftsByTarget.get(target) ?? emptyProfileDraft;
  const { profileId, answers, manualDecisions, evaluation, error, evaluating } = draft;
  const currentSession = profileSession?.target === target ? profileSession : null;
  const { capability: targetCapability, registryScoped, resolving: resolvingTarget, resolutionError, refreshTarget } = useOnboardingTargetCapability(target);
  const supportsProfileSession = targetCapability.profileSession === "owner-scoped" && !resolvingTarget && !resolutionError;
  const profileQuery = useQuery({ queryKey: ["onboarding-profiles", target], queryFn: () => fetchProfiles(target), retry: false, enabled: supportsProfileSession });
  const profiles = profileQuery.data?.profiles ?? [];
  const profilesInitialized = profileQuery.isSuccess;
  const selectedProfile = useMemo(() => profiles.find((profile) => profile.id === profileId), [profiles, profileId]);

  useEffect(() => {
    if (!registryScoped || !profileQuery.isError) return;
    void refreshTarget();
  }, [profileQuery.isError, refreshTarget, registryScoped]);

  useEffect(() => {
    if (!profilesInitialized || profiles.length === 0) return;
    const sessionMarker = currentSession?.revision || JSON.stringify(currentSession?.updatedAt) || "none";
    updateTargetDraft(target, (current) => {
      if (current.appliedSessionMarker === sessionMarker) return current;
      const savedProfile = currentSession?.reconciliationState === "profile_revoked"
      ? ""
      : currentSession?.profileId && profiles.some((profile) => profile.id === currentSession.profileId)
      ? currentSession.profileId
      : profiles.find((profile) => profile.default)?.id ?? profiles[0]?.id ?? "";
      if (current.dirty && currentSession === null) return { ...current, appliedSessionMarker: sessionMarker };
      return {
        ...current,
        profileId: savedProfile,
        answers: currentSession?.answers ?? {},
        manualDecisions: currentSession?.manualDecisions ?? {},
        evaluation: null,
        error: null,
        dirty: false,
        appliedSessionMarker: sessionMarker,
      };
    });
  }, [currentSession, profiles, profilesInitialized, target, updateTargetDraft]);

  useEffect(() => {
    if (!profileId || !supportsProfileSession) return;
    let active = true;
    updateTargetDraft(target, (current) => ({ ...current, evaluating: true }));
	  evaluateProfile(profileId, answers, target, manualDecisions)
      .then((result) => active && updateTargetDraft(target, (current) => ({ ...current, evaluation: result, evaluating: false })))
      .catch(() => active && updateTargetDraft(target, (current) => ({ ...current, error: i18n.t("onboarding.profile.evaluateError"), evaluating: false })))
      .finally(() => active && updateTargetDraft(target, (current) => ({ ...current, evaluating: false })));
    return () => { active = false; };
	}, [answers, manualDecisions, profileId, target, updateTargetDraft, supportsProfileSession]);

  useEffect(() => {
    // A profile catalog can resolve before the parent's durable session read.
    // Do not persist its default selection and erase saved answers in that gap.
    if (!supportsProfileSession || !onProfileSessionChange || !profileId || !selectedProfile || !profilesInitialized || (!draft.dirty && !currentSession)) return;
    const timer = window.setTimeout(() => onProfileSessionChange({
      target,
      mode: "guided",
      profileId,
      profileVersion: selectedProfile.version,
      catalogRevision: currentSession?.catalogRevision,
      consequenceDigest: evaluation?.digest || currentSession?.consequenceDigest,
      baseRevision: currentSession?.baseRevision || profileSessionBaseRevision,
      answers,
	      manualDecisions,
      targetContext: currentSession?.targetContext ?? {},
    }), 250);
    return () => window.clearTimeout(timer);
  }, [answers, manualDecisions, profileId, evaluation?.digest, currentSession, currentSession?.baseRevision, currentSession?.catalogRevision, currentSession?.consequenceDigest, JSON.stringify(currentSession?.targetContext ?? {}), profileSessionBaseRevision, onProfileSessionChange, profilesInitialized, selectedProfile, target, draft.dirty, supportsProfileSession]);

  const updateAnswer = (question: ProfileQuestion, value: unknown) => {
    updateTargetDraft(target, (current) => ({ ...current, error: null, dirty: true, answers: { ...current.answers, [question.id]: value } }));
  };

  if (resolvingTarget) return <section className="welcome-plan" data-testid="profile-surface"><p role="status">{i18n.t("onboarding.profile.checkingTarget")}</p></section>;
  if (resolutionError) return <section className="welcome-plan" data-testid="profile-surface"><p role="alert">{i18n.t("onboarding.profile.targetVerifyError")} {resolutionError instanceof Error ? resolutionError.message : "target list unavailable"}</p>{registryScoped && <button type="button" onClick={() => void refreshTarget()}>{i18n.t("onboarding.profile.checkTargetAgain")}</button>}</section>;
  if (!supportsProfileSession) return <section className="welcome-plan" data-testid="profile-surface"><p role="alert" data-testid="unsupported-profile-target">{targetCapability.reason}</p>{registryScoped && <button type="button" onClick={() => void refreshTarget()}>{i18n.t("onboarding.profile.checkTargetAgain")}</button>}</section>;
  if (profileQuery.isLoading) return <section className="welcome-plan" data-testid="profile-surface"><p role="status">{i18n.t("onboarding.profile.loading")}</p></section>;
  if (profileQuery.isError && profiles.length === 0) return <section className="welcome-plan" data-testid="profile-surface"><p role="alert" data-testid="profile-read-failed">{i18n.t("onboarding.profile.readFailed")}</p><button type="button" onClick={() => void profileQuery.refetch()}>{i18n.t("onboarding.profile.retryRead")}</button><StepPlan target={target} onAccept={onAccept ?? (() => Promise.resolve())} onAdjust={onAdjust} /></section>;

  const questions = evaluation?.questions ?? [];
  const recommended = evaluation?.scenarios ?? [];
  const recommendations = evaluation?.recommendations ?? [];
  const resources = evaluation?.resources ?? [];
  const chooseManual = () => {
    onProfileSessionChange?.({
      target,
      mode: "manual",
      profileId,
      profileVersion: selectedProfile?.version,
      catalogRevision: currentSession?.catalogRevision,
      consequenceDigest: evaluation?.digest || currentSession?.consequenceDigest,
      baseRevision: currentSession?.baseRevision || profileSessionBaseRevision,
      answers,
	      manualDecisions,
      targetContext: currentSession?.targetContext ?? {},
    });
    onAdjust();
  };

  const profileNeedsReview = currentSession?.reconciliationState === "review_required"
    || currentSession?.reconciliationState === "profile_revoked"
    || currentSession?.reconciliationState === "unavailable";

  return <section className="welcome-plan" data-testid="profile-surface">
    {profileQuery.isError && <p role="alert" data-testid="profile-read-failed">{i18n.t("onboarding.profile.refreshFailed")} <button type="button" onClick={() => void profileQuery.refetch()}>{i18n.t("onboarding.profile.retryRead")}</button></p>}
    <div className="section-divider"><span>{i18n.t("onboarding.profile.eyebrow")}</span></div>
    <h2>{i18n.t("onboarding.profile.heading")}</h2>
    <p>{i18n.t("onboarding.profile.description")}</p>
    <div className="block text-sm">
      <span className="block">{i18n.t("onboarding.profile.select")}</span>
      <Select
        aria-label={i18n.t("onboarding.profile.select")}
        data-testid="purpose-profile-select"
        className="mt-1"
        value={profileId}
        onValueChange={(value) => updateTargetDraft(target, (current) => ({ ...current, profileId: value, answers: {}, manualDecisions: {}, evaluation: null, error: null, dirty: true }))}
        options={profiles.map((profile) => ({ value: profile.id, label: profileText(profile.titleKey, profile.id) }))}
      />
    </div>
    {selectedProfile && <p>{profileText(selectedProfile.descriptionKey, selectedProfile.id)}</p>}
    {selectedProfile?.provenanceSource && <p>{i18n.t("onboarding.profile.provenance", { source: selectedProfile.provenanceSource, revision: selectedProfile.provenanceRevision || "unspecified" })}</p>}
    {error && <p role="alert" data-testid="purpose-profile-error">{error}</p>}
    {profileNeedsReview && <div className="profile-session-reconciliation" role="status" data-testid="profile-session-reconciliation">
      <strong>{i18n.t("onboarding.profile.reviewRequired")}</strong>
      <p>{currentSession.reconciliationReasons?.join(" ") || i18n.t("onboarding.profile.reviewDescription")}</p>
      {currentSession.reconciliationChanges && currentSession.reconciliationChanges.length > 0 && <ul data-testid="profile-session-reconciliation-changes">
        {currentSession.reconciliationChanges.map((change) => <li key={`${change.kind}-${change.field}-${change.after}`}>
          <strong>{change.field}</strong>: {change.impact}
        </li>)}
      </ul>}
    </div>}
    {questions.map((question) => <ProfileQuestionInput key={question.id} question={question} value={answers[question.id]} onChange={(value) => updateAnswer(question, value)} />)}
    {evaluating && <p role="status">{i18n.t("onboarding.profile.evaluating")}</p>}
    {evaluation?.issues.map((issue) => <p role="alert" key={`${issue.field}-${issue.code}`}>{issue.message}</p>)}
    {evaluation?.valid && (recommendations.length > 0 || recommended.length > 0 || resources.length > 0) && <Card className="profile-recommendation-card" data-testid="profile-recommendation-summary">
      <CardHeader>
        <CardTitle as="h3">{i18n.t("onboarding.profile.recommendationHeading")}</CardTitle>
        <CardDescription>{i18n.t("onboarding.profile.recommended", { scenarios: recommended.join(", ") || i18n.t("onboarding.profile.noScenarios") })}</CardDescription>
      </CardHeader>
      <CardContent className="profile-recommendation-card__content">
	        {recommendations.length > 0 && <ul className="profile-recommendation-list" aria-label={i18n.t("onboarding.profile.recommendationsLabel")}>
	          {recommendations.map((recommendation) => <li key={recommendation.key || `${recommendation.capabilityRef}-${recommendation.ruleId}`} className="profile-recommendation-list__item">
	            {selectedProfile?.manualSelectionAvailable && <Checkbox
	              checked={recommendation.selected}
	              data-testid={`profile-recommendation-${recommendation.key}`}
	              disabled={recommendation.required}
              onCheckedChange={(checked) => updateTargetDraft(target, (current) => ({ ...current, dirty: true, manualDecisions: { ...current.manualDecisions, [recommendation.key]: checked } }))}
	              label={i18n.t("onboarding.profile.includeRecommendation", { defaultValue: "Include this recommendation" })}
	            />}
            <strong>{recommendation.capabilityRef}</strong>
            <p>{profileText(recommendation.reasonKey, i18n.t("onboarding.profile.reasonUnavailable"))}</p>
            {recommendation.scenarioRefs.length > 0 && <span>{i18n.t("onboarding.profile.supports", { scenarios: recommendation.scenarioRefs.join(", ") })}</span>}
          </li>)}
        </ul>}
        {resources.length > 0 && <p className="profile-recommendation-card__resources"><strong>{i18n.t("onboarding.profile.resources")}</strong> {resources.join(", ")}</p>}
      </CardContent>
    </Card>}
    {profileSessionSaveState !== "idle" && <div className={`profile-session-status profile-session-status--${profileSessionSaveState}`} role={profileSessionError ? "alert" : "status"} data-testid="profile-session-status">
      <span>{profileSessionError ?? (profileSessionSaveState === "saving" ? i18n.t("onboarding.profile.saving") : profileSessionSaveState === "saved" ? i18n.t("onboarding.profile.saved") : i18n.t("onboarding.profile.unsaved"))}</span>
      {(profileSessionSaveState === "failed" || profileSessionSaveState === "conflict") && onRetryProfileSessionSave && <Button type="button" variant="secondary" size="sm" onClick={onRetryProfileSessionSave}>{i18n.t("onboarding.profile.retry")}</Button>}
    </div>}
    <div className="welcome-plan__actions">
      <Button type="button" disabled={!evaluation?.valid || evaluating} onClick={() => { void onAccept?.(profileId, recommended); }} data-testid="purpose-profile-accept">{i18n.t("onboarding.profile.use")}</Button>
      <Button type="button" variant="secondary" onClick={chooseManual} data-testid="purpose-profile-manual">{i18n.t("onboarding.profile.manual")}</Button>
    </div>
  </section>;
}

function ProfileQuestionInput({ question, value, onChange }: { question: ProfileQuestion; value: unknown; onChange: (value: unknown) => void }) {
  if (!question.visible) return null;
  if (question.type === "multi-select") {
    const selected = Array.isArray(value) ? value as string[] : [];
    return <fieldset><legend>{profileText(question.promptKey, question.id)}</legend>{question.options.map((option) => <Checkbox key={option.id} data-testid={`profile-question-${question.id}-${option.id}`} checked={selected.includes(option.id)} onCheckedChange={(checked) => onChange(checked ? [...selected, option.id] : selected.filter((item) => item !== option.id))} label={profileText(option.labelKey, option.id)} />)}</fieldset>;
  }
  if (question.type === "single-select") return <div className="block text-sm"><span className="block">{profileText(question.promptKey, question.id)}</span><Select data-testid={`profile-question-${question.id}`} aria-label={profileText(question.promptKey, question.id)} className="mt-1" value={typeof value === "string" ? value : ""} onValueChange={onChange as (value: string) => void} options={question.options.map((option) => ({ value: option.id, label: profileText(option.labelKey, option.id) }))} placeholder={i18n.t("onboarding.profile.choose")} /></div>;
  if (question.type === "boolean") return <Checkbox data-testid={`profile-question-${question.id}`} checked={value === true} onCheckedChange={onChange} label={profileText(question.promptKey, question.id)} />;
  return <label className="block text-sm">{profileText(question.promptKey, question.id)}<Input data-testid={`profile-question-${question.id}`} className="mt-1" value={typeof value === "string" ? value : ""} onChange={(event) => onChange(event.target.value)} /></label>;
}

function profileText(key: string, fallback: string, options?: Record<string, unknown>) {
  const translatedKey = key.startsWith("onboarding.") ? key : `onboarding.${key}`;
  return i18n.t(translatedKey, { defaultValue: fallback, ...options });
}

function Commitment({ icon, title, description }: { icon: React.ReactNode; title: string; description: string }) {
  return <div className="commitment-row">
    <span className="commitment-row__icon">{icon}</span>
    <div><p>{title}</p><span>{description}</span></div>
  </div>;
}

function formatBytes(value?: number) {
  if (!value) return "—";
  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  let amount = value;
  let unit = 0;
  while (amount >= 1024 && unit < units.length - 1) { amount /= 1024; unit += 1; }
  return `${amount >= 10 || unit === 0 ? Math.round(amount) : amount.toFixed(1)} ${units[unit]}`;
}
