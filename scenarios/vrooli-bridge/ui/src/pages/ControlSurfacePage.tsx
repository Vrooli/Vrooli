import { selectors } from "../consts/selectors";
import { GrantPanel } from "../features/trust/GrantPanel";
import { FleetPanel } from "../features/fleet/FleetPanel";
import { OnboardNodeForm } from "../features/fleet/OnboardNodeForm";
import { RunHistory } from "../features/runs/RunHistory";

type ControlSurfacePageProps = {
  area: "Sessions" | "Rollouts" | "Trust" | "Setup";
  description: string;
};

/** Shared route shell for secondary operator areas. Each route composes a real
 * typed domain panel; the shell supplies only its heading and landmark. */
export function ControlSurfacePage({ area, description }: ControlSurfacePageProps) {
  const pageSelector = {
    Sessions: selectors.pages.sessions,
    Rollouts: selectors.pages.rollouts,
    Trust: selectors.pages.trust,
    Setup: selectors.pages.setup,
  }[area];
  return (
    <section data-testid={pageSelector} aria-labelledby={`${area.toLowerCase()}-heading`} className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <h2 id={`${area.toLowerCase()}-heading`} className="text-2xl font-semibold">{area}</h2>
      <p className="text-app-muted-foreground">{description}</p>
      {area === "Sessions" && <RunHistory />}
      {area === "Rollouts" && <FleetPanel />}
      {area === "Trust" && <GrantPanel />}
      {area === "Setup" && <OnboardNodeForm />}
    </section>
  );
}
