import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { VelocityBoard } from "../features/velocity/VelocityBoard";
import { useTranslation } from "../i18n";

/** Velocity board page. */
export function VelocityPage() {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.pages.velocity}
      aria-labelledby="velocity-heading"
      className="flex flex-col gap-4"
    >
      <header className="flex flex-col gap-1">
        <h2 id="velocity-heading" className="text-2xl font-semibold">
          {t(strings.pages.velocity.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.velocity.description)}</p>
      </header>
      <VelocityBoard />
    </section>
  );
}
