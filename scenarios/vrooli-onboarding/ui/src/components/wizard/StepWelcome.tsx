import { useEffect, useMemo, useRef, useState } from "react";
import { Check, HardDrive, ShieldCheck, Undo2 } from "lucide-react";
import { fetchHostFacts, type HostFacts } from "../../api/host";
import { StatCard } from "@vrooli/react-component-library/StatCard/1";
import { StepPlan } from "./StepPlan";
import { i18n } from "../../i18n";
import { evaluateProfile, fetchProfiles } from "../../api/profiles";
import type { EvaluateProfileResponse, Profile, ProfileQuestion } from "@vrooli/proto-types/vrooli-onboarding/v1/profiles/profiles_pb";
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
  const [profiles, setProfiles] = useState<Profile[]>([]);
	const [profileId, setProfileId] = useState("");
	const [answers, setAnswers] = useState<Record<string, unknown>>({});
	const [manualDecisions, setManualDecisions] = useState<Record<string, boolean>>({});
  const [evaluation, setEvaluation] = useState<EvaluateProfileResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [evaluating, setEvaluating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [profilesInitialized, setProfilesInitialized] = useState(false);
  const appliedSessionMarkerRef = useRef<string | null>(null);
  const selectedProfile = useMemo(() => profiles.find((profile) => profile.id === profileId), [profiles, profileId]);

  useEffect(() => {
    let active = true;
    fetchProfiles(target)
      .then((response) => {
        if (!active) return;
        setProfiles(response.profiles);
        setProfilesInitialized(true);
      })
      .catch(() => active && setError(i18n.t("onboarding.profile.loadError")))
      .finally(() => active && setLoading(false));
    return () => { active = false; };
  }, [target]);

  useEffect(() => {
    if (!profilesInitialized || profiles.length === 0) return;
    const sessionMarker = profileSession?.revision || profileSession?.updatedAt?.toString() || "none";
    if (appliedSessionMarkerRef.current === sessionMarker) return;
    const savedProfile = profileSession?.reconciliationState === "profile_revoked"
      ? ""
      : profileSession?.profileId && profiles.some((profile) => profile.id === profileSession.profileId)
      ? profileSession.profileId
      : profiles.find((profile) => profile.default)?.id ?? profiles[0]?.id ?? "";
    setProfileId(savedProfile);
	const nextAnswers = profileSession?.answers ?? {};
	setAnswers((current) => JSON.stringify(current) === JSON.stringify(nextAnswers) ? current : nextAnswers);
	const nextManualDecisions = profileSession?.manualDecisions ?? {};
	setManualDecisions((current) => JSON.stringify(current) === JSON.stringify(nextManualDecisions) ? current : nextManualDecisions);
    appliedSessionMarkerRef.current = sessionMarker;
  }, [profilesInitialized, profiles, profileSession]);

  useEffect(() => {
    if (!profileId) return;
    let active = true;
    setEvaluating(true);
	  evaluateProfile(profileId, answers, target, manualDecisions)
      .then((result) => active && setEvaluation(result))
      .catch(() => active && setError(i18n.t("onboarding.profile.evaluateError")))
      .finally(() => active && setEvaluating(false));
    return () => { active = false; };
	}, [answers, manualDecisions, profileId, target]);

  useEffect(() => {
    if (!onProfileSessionChange || !profileId || !selectedProfile || !profilesInitialized) return;
    const timer = window.setTimeout(() => onProfileSessionChange({
      target,
      mode: "guided",
      profileId,
      profileVersion: selectedProfile.version,
      catalogRevision: profileSession?.catalogRevision,
      consequenceDigest: evaluation?.digest || profileSession?.consequenceDigest,
      baseRevision: profileSession?.baseRevision || profileSessionBaseRevision,
      answers,
	      manualDecisions,
      targetContext: profileSession?.targetContext ?? {},
    }), 250);
    return () => window.clearTimeout(timer);
  }, [answers, manualDecisions, profileId, evaluation?.digest, profileSession?.baseRevision, profileSession?.catalogRevision, profileSession?.consequenceDigest, JSON.stringify(profileSession?.targetContext ?? {}), profileSessionBaseRevision, onProfileSessionChange, profilesInitialized, selectedProfile, target]);

  const updateAnswer = (question: ProfileQuestion, value: unknown) => {
    setError(null);
    setAnswers((current) => ({ ...current, [question.id]: value }));
  };

  if (loading) return <section className="welcome-plan" data-testid="profile-surface"><p role="status">{i18n.t("onboarding.profile.loading")}</p></section>;
  if (error && profiles.length === 0) return <section className="welcome-plan" data-testid="profile-surface"><p role="alert">{error}</p><StepPlan target={target} onAccept={onAccept ?? (() => Promise.resolve())} onAdjust={onAdjust} /></section>;

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
      catalogRevision: profileSession?.catalogRevision,
      consequenceDigest: evaluation?.digest || profileSession?.consequenceDigest,
      baseRevision: profileSession?.baseRevision || profileSessionBaseRevision,
      answers,
	      manualDecisions,
      targetContext: profileSession?.targetContext ?? {},
    });
    onAdjust();
  };

  const profileNeedsReview = profileSession?.reconciliationState === "review_required"
    || profileSession?.reconciliationState === "profile_revoked"
    || profileSession?.reconciliationState === "unavailable";

  return <section className="welcome-plan" data-testid="profile-surface">
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
	        onValueChange={(value) => { setProfileId(value); setAnswers({}); setManualDecisions({}); setEvaluation(null); }}
        options={profiles.map((profile) => ({ value: profile.id, label: profileText(profile.titleKey, profile.id) }))}
      />
    </div>
    {selectedProfile && <p>{profileText(selectedProfile.descriptionKey, selectedProfile.id)}</p>}
    {selectedProfile?.provenanceSource && <p>{i18n.t("onboarding.profile.provenance", { source: selectedProfile.provenanceSource, revision: selectedProfile.provenanceRevision || "unspecified" })}</p>}
    {error && <p role="alert" data-testid="purpose-profile-error">{error}</p>}
    {profileNeedsReview && <div className="profile-session-reconciliation" role="status" data-testid="profile-session-reconciliation">
      <strong>{i18n.t("onboarding.profile.reviewRequired")}</strong>
      <p>{profileSession?.reconciliationReasons?.join(" ") || i18n.t("onboarding.profile.reviewDescription")}</p>
      {profileSession?.reconciliationChanges && profileSession.reconciliationChanges.length > 0 && <ul data-testid="profile-session-reconciliation-changes">
        {profileSession.reconciliationChanges.map((change) => <li key={`${change.kind}-${change.field}-${change.after}`}>
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
	              onCheckedChange={(checked) => setManualDecisions((current) => ({ ...current, [recommendation.key]: checked }))}
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
