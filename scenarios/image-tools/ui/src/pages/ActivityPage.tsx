import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { ActivityCard } from "../features/activity/ActivityCard";
import { useTranslation } from "../i18n";

/** Activity route — the Console-zone monitor for live and recent operations. */
export function ActivityPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.activity}
      aria-labelledby="activity-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="activity-heading" className="text-2xl font-semibold">
        {t(strings.pages.activity.title)}
      </h2>
      <ActivityCard />
    </section>
  );
}
