import { useCallback, useEffect, useState } from "react";

export type TabType = "dashboard" | "trends" | "timeline" | "incidents" | "docs";

export function getTabFromHash(): TabType {
  const hash = window.location.hash.slice(1);
  if (hash === "trends") return "trends";
  if (hash === "timeline") return "timeline";
  if (hash === "incidents") return "incidents";
  if (hash === "docs" || hash.startsWith("docs?")) return "docs";
  return "dashboard";
}

export function useActiveTab() {
  const [activeTab, setActiveTab] = useState<TabType>(getTabFromHash);

  const handleTabChange = useCallback((tab: TabType) => {
    setActiveTab(tab);
    window.location.hash = tab === "dashboard" ? "" : tab;
  }, []);

  useEffect(() => {
    const handleHashChange = () => {
      setActiveTab(getTabFromHash());
    };
    window.addEventListener("hashchange", handleHashChange);
    return () => window.removeEventListener("hashchange", handleHashChange);
  }, []);

  return { activeTab, handleTabChange };
}
