/**
 * Zustand store for sidebar state management.
 * Handles collapsed state (persisted to localStorage) and active section tracking.
 * Also exports centralized section constants (icons, stage mappings).
 */

import { create } from "zustand";
import { persist } from "zustand/middleware";
import {
  Settings,
  Package,
  ShieldCheck,
  Wand2,
  Hammer,
  TestTube,
  Cloud,
} from "lucide-react";
import type { PipelineStage } from "./pipelineStore";

/** Section identifiers matching pipeline stages + configuration */
export type SectionId =
  | "configuration"
  | "bundle"
  | "preflight"
  | "generate"
  | "build"
  | "smoketest"
  | "deploy";

/** All section IDs in order */
export const SECTION_IDS: SectionId[] = [
  "configuration",
  "bundle",
  "preflight",
  "generate",
  "build",
  "smoketest",
  "deploy",
];

/** Section metadata for display */
export const SECTION_METADATA: Record<SectionId, { label: string; description: string }> = {
  configuration: { label: "Configuration", description: "Set up your desktop app" },
  bundle: { label: "Bundle", description: "Package dependencies" },
  preflight: { label: "Preflight", description: "Validate environment" },
  generate: { label: "Generate", description: "Create wrapper code" },
  build: { label: "Build", description: "Compile installers" },
  smoketest: { label: "Smoke Test", description: "Test artifacts" },
  deploy: { label: "Deploy", description: "Upload to LPBS" },
};

/** Map section ID to icon component */
export const SECTION_ICONS: Record<SectionId, typeof Settings> = {
  configuration: Settings,
  bundle: Package,
  preflight: ShieldCheck,
  generate: Wand2,
  build: Hammer,
  smoketest: TestTube,
  deploy: Cloud,
};

/** Map section ID to pipeline stage (configuration has no stage) */
export const SECTION_TO_STAGE: Partial<Record<SectionId, PipelineStage>> = {
  bundle: "bundle",
  preflight: "preflight",
  generate: "generate",
  build: "build",
  smoketest: "smoketest",
  deploy: "deploy",
};

interface SidebarStoreState {
  /** Whether the sidebar is collapsed (desktop only) */
  collapsed: boolean;
  /** Currently active/visible section */
  activeSection: SectionId;
  /** Whether the mobile drawer is open */
  mobileDrawerOpen: boolean;
}

interface SidebarStoreActions {
  /** Toggle sidebar collapsed state */
  toggleCollapsed: () => void;
  /** Set sidebar collapsed state */
  setCollapsed: (collapsed: boolean) => void;
  /** Set the active section */
  setActiveSection: (section: SectionId) => void;
  /** Set mobile drawer open/closed */
  setMobileDrawerOpen: (open: boolean) => void;
}

type SidebarStore = SidebarStoreState & SidebarStoreActions;

export const useSidebarStore = create<SidebarStore>()(
  persist(
    (set) => ({
      // State
      collapsed: false,
      activeSection: "configuration",
      mobileDrawerOpen: false,

      // Actions
      toggleCollapsed: () => set((state) => ({ collapsed: !state.collapsed })),
      setCollapsed: (collapsed) => set({ collapsed }),
      setActiveSection: (section) => set({ activeSection: section }),
      setMobileDrawerOpen: (open) => set({ mobileDrawerOpen: open }),
    }),
    {
      name: "scenario-to-desktop-sidebar",
      // Only persist collapsed state, not active section
      partialize: (state) => ({ collapsed: state.collapsed }),
    }
  )
);

// Selectors
export const selectCollapsed = (state: SidebarStore) => state.collapsed;
export const selectActiveSection = (state: SidebarStore) => state.activeSection;
