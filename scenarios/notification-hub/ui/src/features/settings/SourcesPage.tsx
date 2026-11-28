import { useState } from "react";
import { Plug } from "lucide-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Select } from "@vrooli/react-component-library/Select/1";
import { Switch } from "@vrooli/react-component-library/Switch/1";
import {
  fetchNotificationCatalog,
  notificationPreferenceControlGroups,
  updateNotificationPreference,
  type MinUrgency,
} from "../../api/preferences";
import { EmptyNotice, QueryError } from "../../components/QueryNotice";
import { SkeletonRows } from "../../components/Skeleton";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { selectors } from "../../consts/selectors";
import { Notice } from "./Notice";
import { SettingsScreen } from "./SettingsScreen";
import { SessionUnavailable } from "../session/SessionGate";
import { useSession } from "../session/useSession";

const URGENCY_LABEL = {
  low: strings.settings.sources.urgency.low,
  normal: strings.settings.sources.urgency.normal,
  high: strings.settings.sources.urgency.high,
  critical: strings.settings.sources.urgency.critical,
} as const satisfies Record<MinUrgency, string>;

/** Which apps may notify you, and the lowest urgency each lets through. Advanced; most people never open it. */
export function SourcesPage() {
  const { t } = useTranslation();
  const session = useSession();
  return (
    <SettingsScreen title={t(strings.settings.section.sources)} description={t(strings.settings.sources.description)} backToSettings>
      {session.isPending ? <SkeletonRows rows={3} /> : session.isError
        ? <SessionUnavailable retry={() => { void session.refetch(); }} />
        : <SourcesContent readOnly={!session.data.signedIn} />}
    </SettingsScreen>
  );
}

function SourcesContent({ readOnly }: { readOnly: boolean }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const queryKey = ["notification-catalog"] as const;
  const catalog = useQuery({ queryKey, queryFn: () => fetchNotificationCatalog() });
  const save = useMutation({
    mutationFn: updateNotificationPreference,
    onSuccess: () => queryClient.invalidateQueries({ queryKey }),
  });
  const [failed, setFailed] = useState<string | null>(null);

  if (catalog.isPending) return <SkeletonRows rows={4} />;
  if (catalog.isError && !catalog.data) {
    return <QueryError error={catalog.error} title={t(strings.settings.sources.loadFailed)} detail={t(strings.settings.sources.loadFailedDetail)} onRetry={() => void catalog.refetch()} retrying={catalog.isFetching} />;
  }
  const { sources, preferences } = catalog.data;
  if (sources.length === 0) {
    return <EmptyNotice icon={<Plug size={28} />} title={t(strings.settings.sources.emptyTitle)} detail={t(strings.settings.sources.emptyDetail)} />;
  }
  const allLabel = t(strings.settings.sources.everything);
  const groups = notificationPreferenceControlGroups(sources, preferences, allLabel, (preference) => {
    if (readOnly) return;
    setFailed(null);
    save.mutate(preference, {
      onError: () => {
        const source = sources.find((item) => item.source_scenario === preference.source_scenario);
        const category = source?.categories.find((item) => item.category === preference.category);
        const label = category?.display_name || category?.category || allLabel;
        setFailed(`${source?.display_name || preference.source_scenario} · ${label}`);
      },
    });
  }).map((group) => ({
    ...group,
    testId: selectors.settings.source({ scenario: group.id }),
    rows: group.rows.map((row) => {
      return {
        ...row,
        testId: selectors.settings.sourceToggle({ scenario: row.sourceScenario, category: row.category }),
      };
    }),
  }));

  return (
    <>
      {readOnly && <Notice>{t(strings.settings.sources.readOnly)}</Notice>}
      {failed !== null && <Notice tone="danger" title={t(strings.settings.sources.saveFailed, { name: failed })}>{t(strings.settings.sources.saveFailedDetail)}</Notice>}
      <div className="grid gap-4">
        {groups.map((group) => (
          <section key={group.id} aria-labelledby={`source-group-${group.id}`} data-testid={group.testId} className="rounded-xl border border-app-border bg-app-surface p-4">
            <h2 id={`source-group-${group.id}`} className="text-base font-semibold">{group.label}</h2>
            <div className="mt-3 grid divide-y divide-app-border">
              {group.rows.map((row) => (
                <div key={row.id} className="grid gap-3 py-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
                  <div className="min-w-0">
                    <p className="font-medium">{row.label}</p>
                    {row.description && <p className="text-sm text-app-muted-foreground">{row.description}</p>}
                  </div>
                  <Switch
                    aria-label={t(strings.settings.sources.allow, { category: row.label })}
                    checked={row.enabled}
                    disabled={readOnly || save.isPending}
                    onCheckedChange={row.onEnabledChange}
                    data-testid={row.testId}
                  />
                  {row.enabled && (
                    <label className="grid gap-1 text-sm sm:col-start-1">
                      <span>{t(strings.settings.sources.minimumUrgency)}</span>
                      <Select
                        aria-label={t(strings.settings.sources.minimumUrgencyFor, { category: row.label })}
                        value={row.minUrgency}
                        disabled={readOnly || save.isPending}
                        options={Object.entries(URGENCY_LABEL).map(([value, label]) => ({ value, label: t(label) }))}
                        onValueChange={row.onMinUrgencyChange}
                      />
                    </label>
                  )}
                </div>
              ))}
            </div>
          </section>
        ))}
      </div>
    </>
  );
}
