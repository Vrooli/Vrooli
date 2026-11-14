import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plug } from "lucide-react";
import { Avatar } from "@vrooli/react-component-library/Avatar/1";
import { Select } from "@vrooli/react-component-library/Select/1";
import { Switch } from "@vrooli/react-component-library/Switch/1";
import {
  MIN_URGENCY_CHOICES,
  fetchNotificationCatalog,
  updateNotificationPreference,
  type CatalogCategory,
  type CatalogSource,
  type MinUrgency,
  type NotificationCatalog,
  type NotificationPreference,
} from "../../api/preferences";
import { EmptyNotice, QueryError } from "../../components/QueryNotice";
import { ListGroup, ListItem } from "../../components/ListRow";
import { SkeletonRows } from "../../components/Skeleton";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { selectors } from "../../consts/selectors";
import { Notice } from "./Notice";
import { OwnerGate } from "./OwnerGate";
import { SettingsScreen } from "./SettingsScreen";

const CATALOG_KEY = ["notification-catalog"] as const;

const URGENCY_LABEL = {
  low: strings.settings.sources.urgency.low,
  normal: strings.settings.sources.urgency.normal,
  high: strings.settings.sources.urgency.high,
  critical: strings.settings.sources.urgency.critical,
} as const satisfies Record<MinUrgency, string>;

/** A preference change, with the name to use if the Hub refuses it. */
interface PreferenceChange {
  preference: NotificationPreference;
  label: string;
}

const prefKey = (source: string, category: string) => `${source}\u0000${category}`;

/**
 * A source that declares no categories only offers one switch for everything it
 * sends. The Hub stores that as the preference with an empty category.
 */
function categoriesOf(source: CatalogSource): CatalogCategory[] {
  if (source.categories.length > 0) return source.categories;
  return [{ category: "", display_name: "", description: "", declared: source.declared, default_enabled: true, default_min_urgency: "low" }];
}

/** Which apps may notify you, and the lowest urgency each lets through. Advanced; most people never open it. */
export function SourcesPage() {
  const { t } = useTranslation();
  return (
    <SettingsScreen title={t(strings.settings.section.sources)} description={t(strings.settings.sources.description)} backToSettings>
      <OwnerGate><SourcesContent /></OwnerGate>
    </SettingsScreen>
  );
}

function SourcesContent() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const catalog = useQuery({ queryKey: CATALOG_KEY, queryFn: fetchNotificationCatalog });
  const [failed, setFailed] = useState<string | null>(null);

  // A change shows at once and is put back if the Hub refuses it, so a row never
  // claims a setting the Hub does not hold.
  const save = useMutation({
    mutationFn: ({ preference }: PreferenceChange) => updateNotificationPreference(preference),
    onMutate: async ({ preference: next }: PreferenceChange) => {
      setFailed(null);
      await queryClient.cancelQueries({ queryKey: CATALOG_KEY });
      const previous = queryClient.getQueryData<NotificationCatalog>(CATALOG_KEY);
      queryClient.setQueryData<NotificationCatalog>(CATALOG_KEY, (current) => current && {
        ...current,
        preferences: [
          ...current.preferences.filter((pref) => prefKey(pref.source_scenario, pref.category) !== prefKey(next.source_scenario, next.category)),
          next,
        ],
      });
      return { previous };
    },
    onError: (_error, { label }, context) => {
      queryClient.setQueryData(CATALOG_KEY, context?.previous);
      setFailed(label);
    },
    onSettled: () => void queryClient.invalidateQueries({ queryKey: CATALOG_KEY }),
  });

  if (catalog.isPending) return <SkeletonRows rows={4} />;
  if (catalog.isError && !catalog.data) {
    return <QueryError error={catalog.error} title={t(strings.settings.sources.loadFailed)} detail={t(strings.settings.sources.loadFailedDetail)} onRetry={() => void catalog.refetch()} retrying={catalog.isFetching} />;
  }
  const { sources, preferences } = catalog.data;
  if (sources.length === 0) {
    return <EmptyNotice icon={<Plug size={28} />} title={t(strings.settings.sources.emptyTitle)} detail={t(strings.settings.sources.emptyDetail)} />;
  }
  const stored = new Map(preferences.map((pref) => [prefKey(pref.source_scenario, pref.category), pref]));

  return (
    <>
      {failed !== null && <Notice tone="danger" title={t(strings.settings.sources.saveFailed, { name: failed })}>{t(strings.settings.sources.saveFailedDetail)}</Notice>}
      {sources.map((source) => {
        const sourceName = source.display_name || source.source_scenario;
        const headingId = `source-${source.source_scenario}`;
        return (
          <section key={source.source_scenario} aria-labelledby={headingId} data-testid={selectors.settings.source({ scenario: source.source_scenario })} className="flex flex-col gap-2">
            <h2 id={headingId} className="flex items-center gap-2.5 px-1 text-base font-semibold [overflow-wrap:anywhere]">
              <span aria-hidden="true"><Avatar name={sourceName} size="xs" shape="rounded" /></span>
              {sourceName}
            </h2>
            <ListGroup>
              {categoriesOf(source).map((category) => {
                const label = category.display_name || category.category || t(strings.settings.sources.everything);
                const pref: NotificationPreference = stored.get(prefKey(source.source_scenario, category.category)) ?? {
                  source_scenario: source.source_scenario,
                  category: category.category,
                  enabled: category.default_enabled,
                  min_urgency: category.default_min_urgency || "low",
                  channel_override: "",
                };
                const change = (patch: Partial<NotificationPreference>) => save.mutate({ preference: { ...pref, ...patch }, label: `${sourceName} · ${label}` });
                return (
                  <ListItem key={category.category} className="flex flex-col gap-3 px-4 py-3">
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex min-w-0 flex-col">
                        <span className="font-medium [overflow-wrap:anywhere]">{label}</span>
                        {category.description && <span className="text-sm text-app-muted-foreground">{category.description}</span>}
                      </div>
                      <span className="inline-flex shrink-0" data-testid={selectors.settings.sourceToggle({ scenario: source.source_scenario, category: category.category })}>
                        <Switch
                          checked={pref.enabled}
                          aria-label={t(strings.settings.sources.allow, { category: label })}
                          onCheckedChange={(enabled) => change({ enabled })}
                        />
                      </span>
                    </div>
                    {pref.enabled && (
                      <label className="flex items-center justify-between gap-3 text-sm">
                        <span className="text-app-muted-foreground">{t(strings.settings.sources.minimumUrgency)}</span>
                        <Select
                          aria-label={t(strings.settings.sources.minimumUrgencyFor, { category: label })}
                          value={pref.min_urgency || "low"}
                          options={MIN_URGENCY_CHOICES.map((choice) => ({ value: choice, label: t(URGENCY_LABEL[choice]) }))}
                          onValueChange={(min_urgency) => change({ min_urgency })}
                        />
                      </label>
                    )}
                  </ListItem>
                );
              })}
            </ListGroup>
          </section>
        );
      })}
    </>
  );
}
