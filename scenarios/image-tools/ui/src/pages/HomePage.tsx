import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { HomeLaunchpad } from "../features/home/HomeLaunchpad";
import { useTranslation } from "../i18n";

/**
 * Home route — the Lume launchpad. The page heading is visually hidden (the
 * launchpad carries its own hero), but present for the document outline / a11y.
 */
export function HomePage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.home}
      aria-labelledby="home-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="home-heading" className="sr-only">
        {t(strings.pages.home.title)}
      </h2>
      <HomeLaunchpad />
    </section>
  );
}
