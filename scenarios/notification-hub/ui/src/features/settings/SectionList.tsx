import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { selectors } from "../../consts/selectors";
import { ListGroup, ListItem, ListRow } from "../../components/ListRow";
import { SECTION_GROUPS, useSettingsSections, type SectionGroup, type SectionKey } from "./sections";
import { useSettingsPath } from "./testContext";

const GROUP_LABEL = {
  reaching: strings.settings.group.reaching,
  app: strings.settings.group.app,
  advanced: strings.settings.group.advanced,
} as const satisfies Record<SectionGroup, string>;

/**
 * The settings, grouped by what they are for, each with where it stands now.
 * On a phone this is the Settings page. Beside an open setting it is the
 * navigation (`compact`): one line per setting, the open one marked, because the
 * open screen already shows where that setting stands.
 */
export function SectionList({ current, compact = false }: { current?: SectionKey; compact?: boolean }) {
  const { t } = useTranslation();
  const path = useSettingsPath();
  const sections = useSettingsSections();
  return (
    <div className="flex flex-col gap-6">
      {SECTION_GROUPS.map((group) => {
        const rows = sections.filter((section) => section.group === group);
        return (
          <section key={group} aria-labelledby={`settings-group-${group}`} className="flex flex-col gap-2">
            <h2 id={`settings-group-${group}`} className="px-1 text-sm font-semibold text-app-muted-foreground">{t(GROUP_LABEL[group])}</h2>
            <ListGroup>
              {rows.map((section) => {
                const Icon = section.icon;
                const active = current === section.key;
                return (
                  <ListItem key={section.key}>
                    <ListRow
                      to={path(section.to)}
                      aria-current={active ? "page" : undefined}
                      data-testid={selectors.settings.row({ key: section.key })}
                      className={active ? "bg-app-primary/10 hover:bg-app-primary/15" : undefined}
                    >
                      <div className={compact ? "flex items-center gap-3" : "flex items-start gap-3"}>
                        <span className={`${compact ? "" : "mt-0.5 "}grid size-8 shrink-0 place-items-center rounded-control bg-app-surface-muted text-app-muted-foreground`} aria-hidden="true">
                          <Icon size={18} />
                        </span>
                        <span className="flex min-w-0 flex-col">
                          <span className="text-base font-semibold">{section.title}</span>
                          {!compact && (section.summary === undefined
                            ? <span aria-hidden="true" className="mt-1 h-4 w-28 rounded bg-app-surface-muted motion-safe:animate-pulse" />
                            : <span className="text-sm text-app-muted-foreground [overflow-wrap:anywhere]">{section.summary}</span>)}
                        </span>
                      </div>
                    </ListRow>
                  </ListItem>
                );
              })}
            </ListGroup>
          </section>
        );
      })}
    </div>
  );
}
