import type { Presentation } from './types';
import type { CropPolicy, PresentationResources } from './resources';
import { assertResourceClosure } from './resourceClosure';

export interface CanonicalWorkspace {
  title: string; group: string; group_label: string; groups: string[];
  sessions_label: string; sessions: string[]; role: string; model: string; reviewer: string; reviewer_model: string;
  branch: string; prompt: string; answer: string; files: string[]; file_label: string; diff: string[];
  command: string; checks: string[]; ready: string; composer: string; return_label: string;
  return_title: string; review_message: string; message_label: string; reply_label: string;
  status: string; today: string; keyboard: string[];
}
export interface CanonicalBackdrop {
  title: string; label: string; selected: string; styles: string[]; surface: string; palette: string;
  panel: string; caption: string; export: string; options: string[]; badge: string; asset_refs: string[];
}
export interface CanonicalWorkflow {
  title: string; label: string; steps: { number: string; title: string; description: string }[];
  browser_title: string; browser_rows: string[]; note: string;
}
export interface CanonicalMonitorMetric {
  id: string; label: string; value: string; unit: string; detail: string; status: string; trend: number[];
}
export interface CanonicalMonitor {
  title: string; host: string; metrics_label: string; metrics: CanonicalMonitorMetric[];
  investigation_label: string; investigation_title: string; investigation_body: string;
  findings: string[]; action_note: string; status: string; uptime: string;
}
export type ConfiguredFixture =
  | { id: string; kind: 'workspace'; workspace: CanonicalWorkspace }
  | { id: string; kind: 'backdrop'; backdrop: CanonicalBackdrop }
  | { id: string; kind: 'workflow'; workflow: CanonicalWorkflow }
  | { id: string; kind: 'monitor'; monitor: CanonicalMonitor };
export interface ResolvedAsset {
  id: string; public_url?: string; width: number; height: number; release_ref: string; content_hash: string;
  mime: string; surface: string; responsive_alternatives?: { surface: string; asset_id: string }[];
  focal_point: { x: number; y: number }; crop_policy: string;
  provenance: { provider: string; job_ref: string; candidate_ref: string };
  overlay_regions: { name: string; x: number; y: number; width: number; height: number;
    measurement: { contrast_ratio: number; minimum_contrast_ratio: number; threshold: number;
      verdict: 'pass' | 'fail' | 'not_measured' } }[];
}

function cropPolicy(value: string): CropPolicy {
  switch (value) {
    case 'center': case 'contain': case 'cover': return value;
    default: throw new Error('Unsupported asset crop policy');
  }
}

/** No fallback fixtures or guessed IDs: canonical arrays are the only content source. */
export function resolveResources(presentation: Presentation): PresentationResources {
  const display = presentation.page.display;
  assertResourceClosure(presentation);
  const visuals: PresentationResources['visuals'] = {};
  const assets: PresentationResources['assets'] = {};
  for (const fixture of presentation.fixtures ?? []) {
    if (visuals[fixture.id]) throw new Error('Duplicate fixture ID');
    const decoration = display.fixture_display[fixture.id];
    switch (fixture.kind) {
      case 'workspace':
        if (!decoration || !('tabs_label' in decoration)) throw new Error(`Missing workspace display labels: ${fixture.id}`);
        visuals[fixture.id] = { ...decoration, ...fixture.workspace, kind: 'workspace' };
        break;
      case 'backdrop':
        if (!decoration) throw new Error(`Missing backdrop display mark: ${fixture.id}`);
        if (fixture.backdrop.styles.length !== fixture.backdrop.asset_refs.length || !fixture.backdrop.styles.includes(fixture.backdrop.selected)) throw new Error('Invalid backdrop selection');
        visuals[fixture.id] = { ...fixture.backdrop, mark: decoration.mark, kind: 'backdrop' };
        break;
      case 'workflow': visuals[fixture.id] = { ...fixture.workflow, kind: 'workflow' }; break;
      case 'monitor':
        if (!decoration) throw new Error(`Missing monitor display mark: ${fixture.id}`);
        if (fixture.monitor.metrics.some(metric => metric.trend.some(sample => !Number.isFinite(sample) || sample < 0 || sample > 1))) throw new Error('Invalid monitor trend sample');
        visuals[fixture.id] = { ...fixture.monitor, mark: decoration.mark, kind: 'monitor' };
        break;
    }
  }
  for (const asset of presentation.assets ?? []) {
    if (assets[asset.id]) throw new Error('Duplicate asset ID');
    const label = display.asset_labels[asset.id];
    const focal = asset.focal_point;
    if (![focal.x, focal.y].every(value => Number.isFinite(value) && value >= 0 && value <= 1)) throw new Error('Invalid asset focal point');
    assets[asset.id] = { src: asset.public_url ?? '', width: asset.width, height: asset.height,
      crop_policy: cropPolicy(asset.crop_policy), focal_point: { ...focal },
      alt: label?.alt ?? '', sizes: label?.sizes, release_ref: asset.release_ref, content_hash: asset.content_hash };
  }
  for (const source of presentation.assets ?? []) {
    const candidates = [source];
    // Width descriptors are safe only for the same composition/crop. Different
    // aspect ratios need explicit art-direction policy, never viewport guessing.
    for (const ref of source.responsive_alternatives ?? []) {
      const candidate = presentation.assets?.find(item => item.id === ref.asset_id);
      if (!candidate) throw new Error('Unresolved responsive asset');
      if (ref.surface === source.surface && candidate.surface === source.surface && candidate.mime === source.mime && candidate.crop_policy === source.crop_policy && candidate.focal_point.x === source.focal_point.x && candidate.focal_point.y === source.focal_point.y && candidate.width * source.height === source.width * candidate.height && !candidates.some(item => item.width === candidate.width)) candidates.push(candidate);
    }
    if (candidates.length > 1) {
      if (candidates.some(item => !item.public_url || /[\s,]/.test(item.public_url))) throw new Error('Unsafe responsive URL');
      const output = assets[source.id];
      if (output?.sizes) output.src_set = candidates.map(item => `${item.public_url ?? ''} ${String(item.width)}w`).join(', ');
    }
  }
  return { shell: display.shell, blocks: display.blocks, apps: display.apps, assets, visuals };
}
