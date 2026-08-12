import { selectors } from "../consts/selectors";

type ControlSurfacePageProps = {
  area: "Sessions" | "Rollouts" | "Trust" | "Setup";
  description: string;
};

/** Shared honest shell for operator areas whose live domain panels are being
 * filled from their typed APIs. It gives every area a stable route and
 * landmark without inventing fake state or hiding unavailable data. */
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
    </section>
  );
}
