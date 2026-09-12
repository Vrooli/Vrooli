import { FileCog, PlayCircle, Settings as SettingsIcon, Target, Wrench } from "lucide-react";
import type { ReactNode } from "react";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ROUTES, ROUTE_PATTERNS } from "../../routes.generated";
import { useGlobalKeydown } from "../hooks";
import { ErrorBoundary } from "../ui/composites/ErrorBoundary";
import { TopHeader } from "./TopHeader";

type AppShellNavItem = {
  id: string;
  href: string;
  patternPrefix: string;
  label: string;
  testId: string;
  icon: typeof Target;
};

export function AppShell(): ReactNode {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const navigate = useNavigate();

  useGlobalKeydown((sequence, event) => {
    if (event.metaKey || event.ctrlKey || event.altKey) return false;
    const destination: Record<string, string> = {
      "g g": ROUTES.goldensIndex,
      "g s": ROUTES.skillsIndex,
      "g m": ROUTES.manifestsIndex,
      "g .": ROUTES.settings,
    };
    const target = destination[sequence];
    if (!target) return false;
    void navigate(target);
    return true;
  });

  const items: readonly AppShellNavItem[] = [
    { id: "goldens", href: ROUTES.goldensIndex, patternPrefix: ROUTE_PATTERNS.goldensIndex, label: t(strings.nav.goldensLabel), testId: selectors.nav.sidebarItemGoldens, icon: Target },
    { id: "skills", href: ROUTES.skillsIndex, patternPrefix: ROUTE_PATTERNS.skillsIndex, label: t(strings.nav.skillsLabel), testId: selectors.nav.sidebarItemSkills, icon: Wrench },
    { id: "manifests", href: ROUTES.manifestsIndex, patternPrefix: ROUTE_PATTERNS.manifestsIndex, label: t(strings.nav.manifestsLabel), testId: selectors.nav.sidebarItemManifests, icon: FileCog },
    { id: "runs", href: ROUTES.runsIndex, patternPrefix: ROUTE_PATTERNS.runsIndex, label: t(strings.nav.runsLabel), testId: selectors.nav.sidebarItemRuns, icon: PlayCircle },
    { id: "settings", href: ROUTES.settings, patternPrefix: ROUTE_PATTERNS.settings, label: t(strings.nav.settingsLabel), testId: selectors.nav.sidebarItemSettings, icon: SettingsIcon },
  ];

  return (
    <LibraryAppShell
      density="sidebar"
      mobileNav="drawer"
      mainMode="scroll"
      brand={<span data-testid={selectors.nav.sidebarLogo}>{t(strings.app.eyebrow)}</span>}
      brandHref={ROUTES.goldensIndex}
      items={items.map(({ id, href, patternPrefix, label, testId, icon: Icon }) => ({
        id,
        href,
        label,
        icon: <Icon size={16} aria-hidden />,
        current: patternPrefix === "/" ? pathname === "/" : pathname.startsWith(patternPrefix),
        testId,
      }))}
      renderLink={(item, props) => (
        <NavLink
          to={props.href}
          end={item.id === "goldens"}
          aria-current={props["aria-current"]}
          aria-disabled={props["aria-disabled"]}
          data-testid={props["data-testid"]}
        >
          {props.children}
        </NavLink>
      )}
      header={<TopHeader />}
      menuLabel={t(strings.nav.menuToggle)}
      closeLabel="Close navigation"
      sidebarStorageKey="development-toolchain-validator.sidebar-width"
      testId={selectors.nav.appShell}
    >
      <ErrorBoundary>
        <Outlet />
      </ErrorBoundary>
    </LibraryAppShell>
  );
}
