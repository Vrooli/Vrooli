import type { LucideIcon } from "lucide-react";
import { ListOrdered, Moon, Palette, Plug, Smartphone, UserRound } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { fetchNotificationCatalog } from "../../api/preferences";
import { strings } from "../../consts/strings";
import { getCurrentLocale, getLocaleConfig, useTranslation } from "../../i18n";
import { THEME_LABEL } from "../../theme/themeChoices";
import { useTheme } from "../../theme/themeContext";
import { useSession } from "../session/useSession";
import { channelLabel } from "./channelLabel";
import { describeDays, describeHours } from "./describeSchedule";
import { devicesQuery, escalationChainQuery, quietWindowsQuery } from "./ownerQueries";
import { groupSchedules } from "./quietSchedule";

export type SectionKey = "reach" | "quietHours" | "escalation" | "appearance" | "session" | "sources";
export type SectionGroup = "reaching" | "app" | "advanced";

export interface SettingsSection {
  key: SectionKey;
  to: string;
  title: string;
  /** `undefined` while the summary is still loading. */
  summary: string | undefined;
  group: SectionGroup;
  icon: LucideIcon;
}

export const SECTION_GROUPS: readonly SectionGroup[] = ["reaching", "app", "advanced"];

/**
 * Every settings screen with the one line that says where it stands now. The
 * summaries read the same cached values the screens do, so a saved change shows
 * up in the list at once.
 */
export function useSettingsSections(): SettingsSection[] {
  const { t, i18n } = useTranslation();
  const lang = i18n.language;
  const { choice } = useTheme();
  const session = useSession();
  const signedIn = Boolean(session.data?.signedIn);
  const owner = signedIn;
  const devices = useQuery({ ...devicesQuery, enabled: owner });
  const windows = useQuery({ ...quietWindowsQuery, enabled: owner });
  const chain = useQuery({ ...escalationChainQuery, enabled: owner });
  const catalog = useQuery({ queryKey: ["notification-catalog"], queryFn: () => fetchNotificationCatalog(), enabled: owner });

  /** A summary for a screen that needs the owner: sign-in prompt, failure, or the value. */
  const ownerSummary = <T,>(query: { data: T | undefined; isError: boolean }, describe: (data: T) => string): string | undefined => {
    if (session.isPending) return undefined;
    if (!owner) return t(strings.settings.summary.signInToView);
    if (query.data !== undefined) return describe(query.data);
    return query.isError ? t(strings.settings.summary.unavailable) : undefined;
  };

  return [
    {
      key: "reach",
      to: "/settings/reach",
      title: t(strings.settings.section.reach),
      group: "reaching",
      icon: Smartphone,
      summary: ownerSummary(devices, (data) => data.devices.length === 0
        ? t(strings.settings.summary.noDevices)
        : t(strings.settings.summary.devices, { count: data.devices.length })),
    },
    {
      key: "quietHours",
      to: "/settings/quiet-hours",
      title: t(strings.settings.section.quietHours),
      group: "reaching",
      icon: Moon,
      summary: ownerSummary(windows, (data) => {
        const schedules = groupSchedules(data.windows);
        const [only] = schedules;
        if (!only) return t(strings.settings.summary.quietOff);
        return schedules.length === 1
          ? t(strings.settings.summary.quietOne, { hours: describeHours(only, lang), days: describeDays(only.days, lang, t) })
          : t(strings.settings.summary.quietMany, { count: schedules.length });
      }),
    },
    {
      key: "escalation",
      to: "/settings/escalation",
      title: t(strings.settings.section.escalation),
      group: "reaching",
      icon: ListOrdered,
      summary: ownerSummary(chain, (data) => data.steps.length === 0
        ? t(strings.settings.summary.escalationNone)
        : data.steps.map((step) => channelLabel(step.channel, t)).join(t(strings.settings.summary.then))),
    },
    {
      key: "appearance",
      to: "/settings/appearance",
      title: t(strings.settings.section.appearance),
      group: "app",
      icon: Palette,
      summary: t(strings.settings.summary.appearance, { theme: t(THEME_LABEL[choice]), language: getLocaleConfig(getCurrentLocale()).nativeLabel }),
    },
    {
      key: "session",
      to: "/settings/session",
      title: t(strings.settings.section.session),
      group: "app",
      icon: UserRound,
      summary: session.isPending
        ? undefined
        : signedIn
          ? t(strings.session.signedInAs, { email: session.data?.email || session.data?.subject || "" })
          : t(strings.settings.summary.signedOut),
    },
    {
      key: "sources",
      to: "/settings/sources",
      title: t(strings.settings.section.sources),
      group: "advanced",
      icon: Plug,
      summary: ownerSummary(catalog, (data) => data.sources.length === 0
        ? t(strings.settings.summary.noSources)
        : t(strings.settings.summary.sources, { count: data.sources.length })),
    },
  ];
}
