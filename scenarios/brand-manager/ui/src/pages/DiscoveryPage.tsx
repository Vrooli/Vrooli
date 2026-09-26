import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { DiscoveryCard } from "../features/discovery/DiscoveryCard";
import { useTranslation } from "../i18n";

export function DiscoveryPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.discovery}
      aria-labelledby="discovery-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="discovery-heading" className="text-2xl font-semibold">
        {t(strings.pages.discovery.title)}
      </h2>
      <DiscoveryCard />
    </section>
  );
}
