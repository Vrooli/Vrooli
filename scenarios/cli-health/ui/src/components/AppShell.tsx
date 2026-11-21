import { Activity, Search, ShieldCheck } from "lucide-react";
import { useState, type ReactNode } from "react";
import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import {
  SUPPORTED_LOCALES,
  getCurrentLocale,
  getLocaleConfig,
  setLocale,
  useTranslation,
} from "../i18n";

export type CliHealthTab = "search" | "validate" | "status";

type Props = {
  children: ReactNode;
  activeTab?: CliHealthTab;
  onTabChange?: (tab: CliHealthTab) => void;
};

const TABS: readonly { id: CliHealthTab; icon: typeof Search; testId: string }[] = [
  { id: "search", icon: Search, testId: selectors.nav.tabSearch },
  { id: "validate", icon: ShieldCheck, testId: selectors.nav.tabValidate },
  { id: "status", icon: Activity, testId: selectors.nav.tabStatus },
];

function Utility() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();

  return (
    <div
      className="flex items-center gap-2"
      role="group"
      aria-label={t(strings.locale.switcherLabel)}
      data-testid={selectors.locale.switcher}
    >
      {SUPPORTED_LOCALES.map((code) => (
        <button
          key={code}
          type="button"
          data-testid={selectors.locale.toggle({ code })}
          onClick={() => void setLocale(code)}
          aria-pressed={currentLocale === code}
          className="rounded-control px-2 py-1 text-xs text-app-muted-foreground hover:text-app-foreground"
        >
          {getLocaleConfig(code).nativeLabel}
        </button>
      ))}
    </div>
  );
}

export function AppShell({ children, activeTab, onTabChange }: Props) {
  const { t } = useTranslation();
  const [internalTab, setInternalTab] = useState<CliHealthTab>("search");
  const selectedTab = activeTab ?? internalTab;

  const selectTab = (tab: CliHealthTab) => {
    setInternalTab(tab);
    onTabChange?.(tab);
  };

  return (
    <LibraryAppShell
      density="sidebar"
      mobileNav="tabs"
      mainMode="scroll"
      brand={<span data-testid={selectors.app.title}>{t(strings.app.title)}</span>}
      brandHref="#search"
      items={TABS.map(({ id, icon: Icon, testId }) => ({
        id,
        href: `#${id}`,
        label: t(strings.nav[id]),
        icon: <Icon size={16} aria-hidden />,
        current: selectedTab === id,
        testId,
      }))}
      renderLink={(item, props) => (
        <a
          href={props.href}
          aria-current={props["aria-current"]}
          data-testid={props["data-testid"]}
          onClick={(event) => {
            event.preventDefault();
            selectTab(item.id as CliHealthTab);
          }}
        >
          {props.children}
        </a>
      )}
      onNavigate={(item) => selectTab(item.id as CliHealthTab)}
      header={
        <div className="min-w-0" data-testid={selectors.nav.root}>
          <p
            data-testid={selectors.app.eyebrow}
            className="text-xs uppercase tracking-wide text-app-muted-foreground"
          >
            {t(strings.app.eyebrow)}
          </p>
          <p data-testid={selectors.app.description} className="text-sm text-app-muted-foreground">
            {t(strings.app.description)}
          </p>
        </div>
      }
      utility={<Utility />}
      sidebarStorageKey="cli-health.sidebar-width"
      testId="cli-health-app-shell"
    >
      {children}
    </LibraryAppShell>
  );
}
