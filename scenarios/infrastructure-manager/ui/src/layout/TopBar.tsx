import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/**
 * Top app bar.
 *
 * There is deliberately no theme control here. `vrooli-annunciator` is a
 * committed single-world design language — see this scenario's `DESIGN.md` —
 * because the lit-lamp metaphor inverts under a light ground, and on this
 * surface an unlit lamp IS the content. A switcher that cannot change anything
 * is a dead affordance, so the control was removed rather than left to lie.
 */
export function TopBar() {
  const { t } = useTranslation();

  return (
    <header
      data-testid={selectors.layout.topBar}
      className="top-chrome flex shrink-0 items-center justify-between gap-4 border-b border-app-border bg-app-shell px-space-sm pb-space-2xs"
    >
      <h1
        data-testid={selectors.app.title}
        className="font-display text-subheading uppercase tracking-[0.14em] text-app-foreground"
      >
        {t(strings.app.title)}
      </h1>
      <p className="font-mono text-body-sm uppercase tracking-[0.18em] text-app-subtle-foreground">
        {t(strings.app.eyebrow)}
      </p>
    </header>
  );
}
