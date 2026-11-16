import { useState } from "react";
import { Plug } from "lucide-react";
import { useNotificationPreferences } from "@vrooli/api-base/notifications/react";
import { NotificationPreferencesPanel } from "@vrooli/react-component-library/AppNotificationSettings/1";
import {
  notificationPreferenceControlGroups,
  notificationPreferencesClient,
  type MinUrgency,
} from "../../api/preferences";
import { EmptyNotice, QueryError } from "../../components/QueryNotice";
import { SkeletonRows } from "../../components/Skeleton";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { selectors } from "../../consts/selectors";
import { Notice } from "./Notice";
import { OwnerGate } from "./OwnerGate";
import { SettingsScreen } from "./SettingsScreen";

const URGENCY_LABEL = {
  low: strings.settings.sources.urgency.low,
  normal: strings.settings.sources.urgency.normal,
  high: strings.settings.sources.urgency.high,
  critical: strings.settings.sources.urgency.critical,
} as const satisfies Record<MinUrgency, string>;

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
  const { catalog, save } = useNotificationPreferences(notificationPreferencesClient);
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
      {failed !== null && <Notice tone="danger" title={t(strings.settings.sources.saveFailed, { name: failed })}>{t(strings.settings.sources.saveFailedDetail)}</Notice>}
      <NotificationPreferencesPanel
        groups={groups}
        disabled={save.isPending}
        strings={{
          minimumUrgency: t(strings.settings.sources.minimumUrgency),
          urgency: {
            low: t(URGENCY_LABEL.low),
            normal: t(URGENCY_LABEL.normal),
            high: t(URGENCY_LABEL.high),
            critical: t(URGENCY_LABEL.critical),
          },
        }}
      />
    </>
  );
}
