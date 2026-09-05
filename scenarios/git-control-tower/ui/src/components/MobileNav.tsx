import { ClipboardCheck, FileText, FileDiff, GitCommit, History } from "lucide-react";
import { BottomNav, type BottomNavItem } from "@vrooli/react-component-library/BottomNav/1";
import type { LayoutSection } from "./LayoutSettingsModal";

interface MobileNavProps {
  activePanel: LayoutSection;
  onPanelChange: (panel: LayoutSection) => void;
  stagedCount?: number;
  unstagedCount?: number;
}

const panels: Array<{ id: LayoutSection; label: string; icon: typeof FileText }> = [
  { id: "changes", label: "Changes", icon: FileText },
  { id: "diff", label: "Diff", icon: FileDiff },
  { id: "commit", label: "Commit", icon: GitCommit },
  { id: "history", label: "History", icon: History },
  { id: "review", label: "Review", icon: ClipboardCheck },
];

export function MobileNav({ activePanel, onPanelChange, stagedCount = 0, unstagedCount = 0 }: MobileNavProps) {
  const totalChanges = stagedCount + unstagedCount;
  const items: BottomNavItem[] = panels.map((panel) => ({
    id: panel.id,
    label: panel.label,
    icon: <panel.icon aria-hidden="true" />,
    active: activePanel === panel.id,
    testId: `mobile-nav-${panel.id}`,
    badge: panel.id === "changes" && totalChanges > 0
      ? { value: totalChanges, max: 99, tone: "warning" as const, label: `${totalChanges} changes` }
      : panel.id === "commit" && stagedCount > 0
        ? { value: stagedCount, max: 99, tone: "success" as const, label: `${stagedCount} staged changes` }
        : undefined,
  }));

  return (
    <BottomNav
      items={items}
      label="Mobile application navigation"
      testId="mobile-nav"
      presentation="flow"
      safeArea="floor"
      activeIndicator="slide"
      onItemSelect={(item) => onPanelChange(item.id as LayoutSection)}
    />
  );
}
