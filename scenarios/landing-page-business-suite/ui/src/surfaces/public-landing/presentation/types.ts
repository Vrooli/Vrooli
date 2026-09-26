/**
 * JSON projection of api/internal/presentation/types.go (owned by Wegener).
 * Keep serialized names here; no route, membership, sorting, or commerce policy.
 * resources.ts projects canonical, locale/page-owned PageDisplay.
 */
import type { ConfiguredFixture, ResolvedAsset } from './resolvedResources';
import type { PresentationDisplay } from './resources';
export type Mode = 'empty' | 'single_app' | 'bundle' | 'app_detail';
export type ActionKind = 'open' | 'download' | 'purchase' | 'request-access' | 'unavailable' | 'anchor' | 'app-detail';
export interface Action {
  kind: ActionKind; label: string; accessible_label: string;
  target?: string; plan_ref?: string; app_key?: string; reason?: string;
}
export interface NavigationItem { label: string; accessible_label: string; target: string }
export interface Footer { label: string; links: NavigationItem[] }
export interface Capability {
  id: string; label: string; benefits: string[]; status: 'available' | 'preview' | 'coming-soon';
  status_label: string; constraints?: string[]; provider_requirements?: string[]; platform_requirements?: string[];
}
export interface Spotlight {
  app_key: string; slug: string; name: string; tagline: string; description: string; detail_route: string;
}
export interface HeroItem { app_key: string; visual_ref: string; exhibit_kind: string; detail_label: string }
export interface DemoPlayback {
  provider: 'youtube' | 'vimeo'; external_url: string; layout: 'stacked' | 'split';
  play_label: string; caption: string; unavailable_label: string;
}
export interface ArtifactExample {
  id: string; kind: 'plan' | 'image' | 'html-preview' | 'video' | 'audio' | 'code' | 'pdf';
  filename: string; caption: string; alt_text: string; width: number; height: number;
  source_ref: string; preview_ref?: string;
  label: string; type: string; title: string; body: string; icon: string;
  steps?: string[]; checks?: string[]; asset_ref?: string; brand?: string; accent?: string;
  frames?: { asset_ref: string; time: string; label: string }[];
  source: { ref: string; media_ref?: string; text_excerpt?: string; integrity_hash?: string; isolation?: string };
}
export interface ContentByKind {
  'product-hero': { app_key: string; eyebrow: string; title: string; description: string; visual_ref: string; fixture_ref?: string; accessibility_label: string; actions: Action[] };
  'bundle-hero': { eyebrow: string; title: string; description: string; accessibility_label: string; hero_items: HeroItem[]; actions: Action[] };
  'capability-strip': { heading: string; items: { capability_id: string; label: string; description: string }[] };
  'product-story': { heading: string; body: string; items: { title: string; description: string; visual_ref?: string; alt_text?: string }[] };
  'product-demo': { heading: string; description: string; renderer_ref: string; fixture_ref?: string; poster_ref?: string; media_ref?: string; alt_text: string; playback?: DemoPlayback };
  'app-spotlights': { heading: string; app_keys: string[]; detail_link_label: string };
  'artifact-explorer': { heading: string; capability_id: string; examples: ArtifactExample[]; selected_example_id: string };
  'voice-story': {
    eyebrow?: string; heading: string; body: string;
    features: { title: string; description: string; capability_id: string }[];
    note: string; input_label: string; transcript: string; summary_label: string; summary_title: string;
    summary_items: string[]; output_label: string; demo_note: string;
    provider_qualification: string; waveform: number[]; capability_ids: string[];
  };
  'device-story': { heading: string; description: string; device: string; visual_ref: string; alt_text: string };
  'capability-roadmap': { heading: string; capability_ids: string[]; status_label: string; description: string };
  'pricing': { heading: string; description: string; plan_refs: string[]; actions: Action[] };
  'closing-action': { heading: string; description: string; actions: Action[]; visual_ref?: string };
  'faq': { heading: string; items: { question: string; answer: string; accessible_label: string }[] };
  'footer': Footer;
}
export type BlockKind = keyof ContentByKind;
export type Block<K extends BlockKind = BlockKind> = {
  [P in K]: { id: string; kind: P; version: number; variant: string; content: ContentByKind[P] }
}[K];
export interface Page {
  display: PresentationDisplay;
  id: string; locale: string; title: string; description: string;
  theme: { variant: 'signal' | 'studio'; primary: string; background: string; accent: string };
  navigation: { label: string; items: NavigationItem[] }; blocks: Block[]; footer: Footer;
}
export interface Presentation {
  schema_version: number; mode: Mode; scope: 'app' | 'bundle'; app_key?: string;
  page: Page; selected_app_keys: string[]; spotlights?: Spotlight[]; capabilities?: Capability[];
  fixtures?: ConfiguredFixture[]; assets?: ResolvedAsset[];
  diagnostics: { preview: boolean; fallback?: boolean; eligible_app_keys: string[]; noindex: boolean; no_store: boolean; resolved_revision: string };
}
export type ResolvedAction =
  | { status: 'ready'; href: string; onActivate?: never }
  | { status: 'ready'; onActivate: () => void; href?: never }
  | { status: 'unavailable'; reason: string };
export type ResolvedActions = Readonly<Record<string, ResolvedAction>>;
/** Stable owner-join identity, not a URL inferred from marketing copy. */
export const actionKey = (action: Action): string =>
  JSON.stringify([action.kind, action.app_key ?? '', action.plan_ref ?? '', action.target ?? '']);
