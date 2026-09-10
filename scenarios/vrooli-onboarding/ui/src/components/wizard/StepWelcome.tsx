import { useEffect, useMemo, useState } from "react";
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

type WelcomeProps = {
  onAccept?: (profile?: string, scenarios?: string[]) => Promise<void>;
  onAdjust?: () => void;
};

export function StepWelcome({ onAccept, onAdjust }: WelcomeProps = {}) {
  const [hostFacts, setHostFacts] = useState<HostFacts | null>(null);

  useEffect(() => {
    let active = true;
    fetchHostFacts().then((facts) => {
      if (active) setHostFacts(facts);
    }).catch(() => {
      if (active) setHostFacts({ available: false });
    });
    return () => { active = false; };
  }, []);

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
      {onAccept && onAdjust && <PurposeProfile onAccept={onAccept} onAdjust={onAdjust} />}
    </div>
  );
}

function PurposeProfile({ onAccept, onAdjust }: { onAccept: WelcomeProps["onAccept"]; onAdjust: () => void }) {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [profileId, setProfileId] = useState("");
  const [answers, setAnswers] = useState<Record<string, unknown>>({});
  const [evaluation, setEvaluation] = useState<EvaluateProfileResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [evaluating, setEvaluating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const selectedProfile = useMemo(() => profiles.find((profile) => profile.id === profileId), [profiles, profileId]);

  useEffect(() => {
    let active = true;
    fetchProfiles()
      .then((response) => {
        if (!active) return;
        setProfiles(response.profiles);
        setProfileId(response.profiles[0]?.id ?? "");
      })
      .catch(() => active && setError(i18n.t("onboarding.profile.loadError")))
      .finally(() => active && setLoading(false));
    return () => { active = false; };
  }, []);

  useEffect(() => {
    if (!profileId) return;
    let active = true;
    setEvaluating(true);
    evaluateProfile(profileId, answers)
      .then((result) => active && setEvaluation(result))
      .catch(() => active && setError(i18n.t("onboarding.profile.evaluateError")))
      .finally(() => active && setEvaluating(false));
    return () => { active = false; };
  }, [answers, profileId]);

  const updateAnswer = (question: ProfileQuestion, value: unknown) => {
    setError(null);
    setAnswers((current) => ({ ...current, [question.id]: value }));
  };

  if (loading) return <section className="welcome-plan" data-testid="profile-surface"><p role="status">{i18n.t("onboarding.profile.loading")}</p></section>;
  if (error && profiles.length === 0) return <section className="welcome-plan" data-testid="profile-surface"><p role="alert">{error}</p><StepPlan onAccept={onAccept ?? (() => Promise.resolve())} onAdjust={onAdjust} /></section>;

  const questions = evaluation?.questions ?? [];
  const recommended = evaluation?.scenarios ?? [];
  return <section className="welcome-plan" data-testid="profile-surface">
    <div className="section-divider"><span>{i18n.t("onboarding.profile.eyebrow")}</span></div>
    <h2>{i18n.t("onboarding.profile.heading")}</h2>
    <p>{i18n.t("onboarding.profile.description")}</p>
    <label className="block text-sm">
      {i18n.t("onboarding.profile.select")}
      <Select
        aria-label={i18n.t("onboarding.profile.select")}
        data-testid="purpose-profile-select"
        className="mt-1"
        value={profileId}
        onValueChange={(value) => { setProfileId(value); setAnswers({}); setEvaluation(null); }}
        options={profiles.map((profile) => ({ value: profile.id, label: profileText(profile.titleKey, profile.id) }))}
      />
    </label>
    {selectedProfile && <p>{profileText(selectedProfile.descriptionKey, selectedProfile.id)}</p>}
    {selectedProfile?.provenanceSource && <p>{i18n.t("onboarding.profile.provenance", { source: selectedProfile.provenanceSource, revision: selectedProfile.provenanceRevision || "unspecified" })}</p>}
    {questions.map((question) => <ProfileQuestionInput key={question.id} question={question} value={answers[question.id]} onChange={(value) => updateAnswer(question, value)} />)}
    {evaluating && <p role="status">{i18n.t("onboarding.profile.evaluating")}</p>}
    {evaluation?.issues.map((issue) => <p role="alert" key={`${issue.field}-${issue.code}`}>{issue.message}</p>)}
    {recommended.length > 0 && <p>{i18n.t("onboarding.profile.recommended", { scenarios: recommended.join(", ") })}</p>}
    <div className="welcome-plan__actions">
      <Button type="button" disabled={!evaluation?.valid || evaluating} onClick={() => { void onAccept?.(profileId, recommended); }} data-testid="purpose-profile-accept">{i18n.t("onboarding.profile.use")}</Button>
      <Button type="button" variant="secondary" onClick={onAdjust} data-testid="purpose-profile-manual">{i18n.t("onboarding.profile.manual")}</Button>
    </div>
    <StepPlan onAccept={onAccept ?? (() => Promise.resolve())} onAdjust={onAdjust} />
  </section>;
}

function ProfileQuestionInput({ question, value, onChange }: { question: ProfileQuestion; value: unknown; onChange: (value: unknown) => void }) {
  if (!question.visible) return null;
  if (question.type === "multi-select") {
    const selected = Array.isArray(value) ? value as string[] : [];
    return <fieldset><legend>{profileText(question.promptKey, question.id)}</legend>{question.options.map((option) => <Checkbox key={option.id} data-testid={`profile-question-${question.id}-${option.id}`} checked={selected.includes(option.id)} onCheckedChange={(checked) => onChange(checked ? [...selected, option.id] : selected.filter((item) => item !== option.id))} label={profileText(option.labelKey, option.id)} />)}</fieldset>;
  }
  if (question.type === "single-select") return <label className="block text-sm">{profileText(question.promptKey, question.id)}<Select data-testid={`profile-question-${question.id}`} aria-label={profileText(question.promptKey, question.id)} className="mt-1" value={typeof value === "string" ? value : ""} onValueChange={onChange as (value: string) => void} options={question.options.map((option) => ({ value: option.id, label: profileText(option.labelKey, option.id) }))} placeholder={i18n.t("onboarding.profile.choose")} /></label>;
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
