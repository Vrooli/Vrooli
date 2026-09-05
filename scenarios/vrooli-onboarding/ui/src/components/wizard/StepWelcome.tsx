import { useEffect, useState } from "react";
import { Check, HardDrive, ShieldCheck, Undo2 } from "lucide-react";
import { fetchV2HostFacts } from "../../lib/api";
import type { V2HostFactsResponse } from "../../types";
import { StatCard } from "@vrooli/react-component-library/StatCard/1";
import { StepPlan } from "./StepPlan";
import { i18n } from "../../i18n";

export function StepWelcome({ onAccept, onAdjust }: { onAccept?: () => Promise<void>; onAdjust?: () => void } = {}) {
  const [hostFacts, setHostFacts] = useState<V2HostFactsResponse | null>(null);

  useEffect(() => {
    let active = true;
    fetchV2HostFacts().then((facts) => {
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
      {onAccept && onAdjust && <section className="welcome-plan" data-testid="plan-surface"><div className="section-divider"><span>{i18n.t("onboarding.welcome.recommended")}</span></div><StepPlan onAccept={onAccept} onAdjust={onAdjust} /></section>}
    </div>
  );
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
