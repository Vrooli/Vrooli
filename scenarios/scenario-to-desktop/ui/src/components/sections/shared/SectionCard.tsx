/**
 * Unified section card wrapper with optional pipeline integration.
 * Supports both pipeline sections (with status) and simple sections.
 */

import {
  forwardRef,
  useState,
  type ReactNode,
  type HTMLAttributes,
} from "react";
import type { LucideIcon } from "lucide-react";
import { Card } from "../../ui/card";
import { SectionHeader } from "./SectionHeader";
import {
  useSidebarStore,
  SECTION_TO_STAGE,
  SECTION_ICONS,
  type SectionId,
} from "../../../store/sidebarStore";
import { usePipelineStore, selectStageStatus } from "../../../store";
import { useIsMobile } from "../../../hooks/useMediaQuery";
import { cn } from "../../../lib/utils";
import { StageStatus } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

/** Status-based border colors for pipeline variant */
const STATUS_BORDER_STYLES: Partial<Record<StageStatus, string>> = {
  [StageStatus.PENDING]: "border-slate-700/50",
  [StageStatus.RUNNING]: "border-blue-500/50 shadow-blue-500/10 shadow-lg",
  [StageStatus.COMPLETED]: "border-green-500/30",
  [StageStatus.FAILED]: "border-red-500/30",
  [StageStatus.SKIPPED]: "border-slate-700/50",
};

interface SectionCardProps extends HTMLAttributes<HTMLDivElement> {
  /** Section title */
  title: string;
  /** Card content */
  children: ReactNode;
  /** Optional icon (auto-resolved from sectionId for pipeline variant) */
  icon?: LucideIcon;
  /** Optional subtitle/description */
  subtitle?: string;
  /** Card variant - 'pipeline' integrates with pipeline store, 'simple' is standalone */
  variant?: "pipeline" | "simple";
  /** Section identifier (required for pipeline variant, optional otherwise) */
  sectionId?: SectionId;
  /** Whether to show the header (default: true) */
  showHeader?: boolean;
  /** Whether the section is collapsible (default: false) */
  collapsible?: boolean;
  /** Initial collapsed state (default: false) */
  defaultCollapsed?: boolean;
  /** Additional className for the content area */
  contentClassName?: string;
}

export const SectionCard = forwardRef<HTMLDivElement, SectionCardProps>(
  (
    {
      title,
      children,
      icon,
      subtitle,
      variant = "simple",
      sectionId,
      showHeader = true,
      collapsible = false,
      defaultCollapsed = false,
      contentClassName,
      className,
      ...props
    },
    ref,
  ) => {
    const [collapsed, setCollapsed] = useState(defaultCollapsed);
    const isMobile = useIsMobile();

    // Pipeline-specific state (only used when variant='pipeline' and sectionId is provided)
    const activeSection = useSidebarStore((s) => s.activeSection);
    const isActive =
      variant === "pipeline" && sectionId ? activeSection === sectionId : false;

    // Get stage status for pipeline sections
    const stage = sectionId ? SECTION_TO_STAGE[sectionId] : undefined;
    const stageStatus = usePipelineStore(
      stage ? selectStageStatus(stage) : () => null,
    );

    // Only apply status styling for pipeline variant with non-configuration sections
    const isPipelineWithStatus =
      variant === "pipeline" && sectionId && sectionId !== "configuration";
    const status = isPipelineWithStatus
      ? (stageStatus ?? StageStatus.PENDING)
      : null;
    const borderStyle = status ? STATUS_BORDER_STYLES[status] : "";

    // Resolve icon: explicit prop > sectionId lookup > undefined
    const resolvedIcon =
      icon ?? (sectionId ? SECTION_ICONS[sectionId] : undefined);

    const handleToggleCollapse = () => {
      setCollapsed((prev) => !prev);
    };

    return (
      <Card
        ref={ref}
        className={cn(
          "transition-all duration-200",
          borderStyle,
          isActive && "ring-2 ring-blue-500/20",
          className,
        )}
        {...props}
      >
        {showHeader && (
          <SectionHeader
            title={title}
            subtitle={subtitle}
            icon={resolvedIcon}
            sectionId={sectionId}
            variant={variant}
            collapsible={collapsible}
            collapsed={collapsed}
            onToggleCollapse={handleToggleCollapse}
          />
        )}
        {/* Content area with internal padding */}
        {!collapsed && (
          <div
            className={cn(
              isMobile ? "p-2.5" : "p-4",
              showHeader && "pt-0",
              contentClassName,
            )}
          >
            {children}
          </div>
        )}
      </Card>
    );
  },
);

SectionCard.displayName = "SectionCard";
