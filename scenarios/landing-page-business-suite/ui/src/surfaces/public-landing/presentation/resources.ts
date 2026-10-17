/**
 * Finite locale/page-owned vocabulary from Go PageDisplay and generated protocol.
 * No fixture content, asset URLs, membership selection, or commerce policy.
 */
import type { Action } from './types';
import type { CanonicalWorkspace, CanonicalBackdrop, CanonicalWorkflow, CanonicalMonitor } from './resolvedResources';
export type Mark = 'letter-a' | 'landscape' | 'suite' | 'play' | 'pulse';
export type CropPolicy = 'center' | 'contain' | 'cover';
export interface MediaAsset {
  src: string; width: number; height: number; alt: string;
  src_set?: string; sizes?: string; release_ref?: string; content_hash?: string;
  crop_policy: CropPolicy; focal_point: { x: number; y: number };
}
export interface WorkspaceDisplay {
  mark: Mark; avatar: string; time: string; tabs_label: string;
  terminal_label: string; messages_label: string; file_changes: Record<string, string>;
}
export type WorkspaceFixture = CanonicalWorkspace & WorkspaceDisplay & { kind: 'workspace' };
export type ArtStudiesFixture = CanonicalBackdrop & { kind: 'backdrop'; mark: Mark };
export type MonitorVisualFixture = CanonicalMonitor & { kind: 'monitor'; mark: Mark };
export type VisualFixture = WorkspaceFixture | ArtStudiesFixture | MonitorVisualFixture | (CanonicalWorkflow & { kind: 'workflow' });
export interface BlockDisplay {
  eyebrow?: string; description?: string; note?: string; accessibility_label?: string;
  badge?: string; mark?: Mark; formats?: string[]; anchors?: Record<string, string>;
  /** Character offsets in canonical text; no duplicated headline copy. */
  heading_breaks?: number[];
  /** Explicit native fixture alternative to the block's released visual. */
  fixture_ref?: string; hero_fixture_refs?: Record<string, string>;
}
export interface PresentationDisplay {
  shell: {
    brand_name: string; brand_mark: Mark; brand_target: string; brand_subtitle?: string;
    skip_label: string; menu_label: string; footer_brand_name: string; footer_brand_mark: Mark;
    footer_brand_target: string; footer_tagline: string; copyright: string; footer_note: string;
    unavailable_reason: string; preview_label: string; header_action?: Action;
    /** Optional same-origin image that replaces the finite SVG brand mark. */
    brand_logo?: string; brand_logo_alt?: string; footer_brand_logo?: string;
  };
  asset_labels: Record<string, { alt: string; sizes?: string }>;
  fixture_display: Record<string, WorkspaceDisplay | { mark: Mark }>;
  blocks: Record<string, BlockDisplay>;
  apps: Record<string, { fixture_ref?: string; visual_ref?: string; mark: Mark; tone: 'amber' | 'sage'; detail_label: string; logo?: string; logo_alt?: string }>;
}
/** Internal reference indexes, built only from sanitized canonical arrays. */
export interface PresentationResources extends Pick<PresentationDisplay, 'shell' | 'blocks' | 'apps'> {
  assets: Record<string, MediaAsset>;
  visuals: Record<string, VisualFixture>;
}
