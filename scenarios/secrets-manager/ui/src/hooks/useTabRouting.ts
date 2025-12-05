import { useLocation, useNavigate } from "react-router-dom";
import { useCallback, useMemo } from "react";

export type ExperienceTab = "dashboard" | "resources" | "compliance" | "deployment";
export type ResourceSubTab = "tier" | "resource";

interface TabRoutingState {
  activeTab: ExperienceTab;
  resourceTab: ResourceSubTab;
}

const VALID_TABS: ExperienceTab[] = ["dashboard", "resources", "compliance", "deployment"];
const VALID_RESOURCE_TABS: ResourceSubTab[] = ["tier", "resource"];

function parseTabFromPath(pathname: string): TabRoutingState {
  const segments = pathname.split("/").filter(Boolean);
  const tabSegment = segments[0] || "dashboard";
  const subTabSegment = segments[1];

  const activeTab = VALID_TABS.includes(tabSegment as ExperienceTab)
    ? (tabSegment as ExperienceTab)
    : "dashboard";

  const resourceTab =
    activeTab === "resources" && VALID_RESOURCE_TABS.includes(subTabSegment as ResourceSubTab)
      ? (subTabSegment as ResourceSubTab)
      : "tier";

  return { activeTab, resourceTab };
}

export function useTabRouting() {
  const location = useLocation();
  const navigate = useNavigate();

  const { activeTab, resourceTab } = useMemo(
    () => parseTabFromPath(location.pathname),
    [location.pathname]
  );

  const setActiveTab = useCallback(
    (tab: ExperienceTab) => {
      if (tab === "resources") {
        navigate(`/${tab}/tier`);
      } else {
        navigate(`/${tab}`);
      }
    },
    [navigate]
  );

  const setResourceTab = useCallback(
    (subTab: ResourceSubTab) => {
      navigate(`/resources/${subTab}`);
    },
    [navigate]
  );

  return {
    activeTab,
    resourceTab,
    setActiveTab,
    setResourceTab
  };
}
