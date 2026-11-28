import type { TemplateNode, TemplatePalette } from "./templates/spec";

/**
 * Shared visual model used by the workspace studio renderer and editing tools.
 * These types describe an editable workspace composition; they do not imply a
 * separate persisted design entity or transport.
 */
export type DesignDevice = "desktop" | "tablet" | "phone";

export type DesignDisplay = "sans" | "serif";

/** Workspace-owned chrome. An empty object means to use library defaults. */
export interface DesignOptions {
  radius?: number;
  display?: DesignDisplay;
}

export interface DesignFrame {
  id: string;
  name: string;
  device: DesignDevice;
  spec: TemplateNode;
}

/** One editable page projection used by shared studio components. */
export interface DesignDocument {
  id: string;
  name: string;
  updatedAt: string;
  revision: number;
  frames: DesignFrame[];
  theme: TemplatePalette;
  options?: DesignOptions;
  scenario?: string | null;
  page?: string | null;
}

/** Input shape for composition helpers; persistence is workspace-owned. */
export interface DesignCompositionInput {
  name: string;
  frames: DesignFrame[];
  theme: TemplatePalette;
  options?: DesignOptions;
  scenario?: string | null;
  page?: string | null;
}
