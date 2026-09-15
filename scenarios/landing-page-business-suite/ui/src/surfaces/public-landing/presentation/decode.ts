import { fromJsonString } from '@bufbuild/protobuf';
import { ResolvedProductPresentationSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';
import type * as Wire from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';
import type { Presentation, Block, ContentByKind, Action, ArtifactExample, HeroItem, NavigationItem, Footer, Capability, Spotlight } from './types';
import type { PresentationDisplay, Mark } from './resources';
import type { ConfiguredFixture, CanonicalWorkspace, CanonicalBackdrop, CanonicalWorkflow, ResolvedAsset } from './resolvedResources';
import { assertRendererContract } from './contract';
import { resolveResources } from './resolvedResources';

function required<T>(value: T | undefined, field: string): T {
  if (value === undefined) throw new Error(`Missing ${field}`);
  return value;
}
function oneOf<T extends string>(value: string, choices: readonly T[]): T {
  const match = choices.find(choice => choice === value);
  if (match === undefined) throw new Error('Unsupported presentation value');
  return match;
}
function unreachable(value: never): never {
  throw new Error(`Unsupported presentation discriminator: ${String(value)}`);
}
/** Reject unrecognized binary fields recursively; strict generated JSON handles text. */
function rejectUnknown(value: unknown): void {
  if (!value || typeof value !== 'object') return;
  if ('$unknown' in value && Array.isArray(value.$unknown) && value.$unknown.length) throw new Error('Unknown presentation fields');
  for (const child of Object.values(value)) rejectUnknown(child);
}
const mark = (value: string): Mark => oneOf<Mark>(value, ['letter-a', 'landscape', 'suite', 'play']);
const mapValues = <T, U>(values: Record<string, T>, map: (value: T) => U): Record<string, U> =>
  Object.fromEntries(Object.entries(values).map(([key, value]) => [key, map(value)]));

function navigationItem(v: Wire.PresentationNavigationItem): NavigationItem {
  return {
    label: v.label,
    accessible_label: v.accessibleLabel,
    target: v.target,
  };
}

function footer(v: Wire.PresentationFooter): Footer {
  return {
    label: v.label,
    links: v.links.map(navigationItem),
  };
}

function action(v: Wire.PresentationAction): Action {
  if (!v.label.trim() || !v.accessibleLabel.trim()) throw new Error('Missing configured action label');
  return {
    kind: oneOf<Action['kind']>(v.kind, ['open', 'download', 'purchase', 'request-access', 'unavailable', 'anchor', 'app-detail']),
    label: v.label,
    accessible_label: v.accessibleLabel,
    target: v.target,
    plan_ref: v.planRef,
    app_key: v.appKey,
    reason: v.reason,
  };
}

function heroItem(v: Wire.PresentationHeroItem): HeroItem {
  return {
    app_key: v.appKey,
    visual_ref: v.visualRef,
    exhibit_kind: v.exhibitKind,
    detail_label: v.detailLabel,
  };
}

function productHero(v: Wire.PresentationProductHero): ContentByKind['product-hero'] {
  return {
    app_key: v.appKey,
    eyebrow: v.eyebrow,
    title: v.title,
    description: v.description,
    visual_ref: v.visualRef,
    fixture_ref: v.fixtureRef,
    accessibility_label: v.accessibilityLabel,
    actions: v.actions.map(action),
  };
}

function bundleHero(v: Wire.PresentationBundleHero): ContentByKind['bundle-hero'] {
  return {
    eyebrow: v.eyebrow,
    title: v.title,
    description: v.description,
    accessibility_label: v.accessibilityLabel,
    hero_items: v.heroItems.map(heroItem),
    actions: v.actions.map(action),
  };
}

function capabilityItem(v: Wire.PresentationCapabilityItem) {
  return {
    capability_id: v.capabilityId,
    label: v.label,
    description: v.description,
  };
}

function capabilityStrip(v: Wire.PresentationCapabilityStrip): ContentByKind['capability-strip'] {
  return {
    heading: v.heading,
    items: v.items.map(capabilityItem),
  };
}

function storyItem(v: Wire.PresentationStoryItem) {
  return {
    title: v.title,
    description: v.description,
    visual_ref: v.visualRef,
    alt_text: v.altText,
  };
}

function productStory(v: Wire.PresentationProductStory): ContentByKind['product-story'] {
  return {
    heading: v.heading,
    body: v.body,
    items: v.items.map(storyItem),
  };
}

function productDemo(v: Wire.PresentationProductDemo): ContentByKind['product-demo'] {
  return {
    heading: v.heading,
    description: v.description,
    renderer_ref: v.rendererRef,
    fixture_ref: v.fixtureRef,
    poster_ref: v.posterRef,
    media_ref: v.mediaRef,
    alt_text: v.altText,
    playback: v.playback ? {
      provider: oneOf(v.playback.provider, ['youtube', 'vimeo']), external_url: v.playback.externalUrl,
      layout: oneOf(v.playback.layout, ['stacked', 'split']), play_label: v.playback.playLabel,
      caption: v.playback.caption, unavailable_label: v.playback.unavailableLabel,
    } : undefined,
  };
}

function appSpotlights(v: Wire.PresentationAppSpotlights): ContentByKind['app-spotlights'] {
  return {
    heading: v.heading,
    app_keys: [...v.appKeys],
    detail_link_label: v.detailLinkLabel,
  };
}

function artifactSource(v: Wire.PresentationArtifactSource) {
  return {
    ref: v.ref,
    media_ref: v.mediaRef,
    text_excerpt: v.textExcerpt,
    integrity_hash: v.integrityHash,
    isolation: v.isolation,
  };
}

function artifactFrame(v: Wire.PresentationArtifactFrame) {
  return {
    asset_ref: v.assetRef,
    time: v.time,
    label: v.label,
  };
}

function artifactExample(v: Wire.PresentationArtifactExample): ArtifactExample {
  return {
    id: v.id,
    kind: oneOf<ArtifactExample['kind']>(v.kind, ['plan', 'image', 'html-preview', 'video', 'audio', 'code', 'pdf']),
    label: v.label,
    filename: v.filename,
    type: v.type,
    title: v.title,
    body: v.body,
    steps: [...v.steps],
    checks: [...v.checks],
    caption: v.caption,
    alt_text: v.altText,
    icon: v.icon,
    width: v.width,
    height: v.height,
    source_ref: v.sourceRef,
    source: artifactSource(required(v.source, 'source')),
    asset_ref: v.assetRef,
    brand: v.brand,
    accent: v.accent,
    frames: v.frames.map(artifactFrame),
    preview_ref: v.previewRef,
  };
}

function artifactExplorer(v: Wire.PresentationArtifactExplorer): ContentByKind['artifact-explorer'] {
  return {
    heading: v.heading,
    capability_id: v.capabilityId,
    examples: v.examples.map(artifactExample),
    selected_example_id: v.selectedExampleId,
  };
}

function voiceFeature(v: Wire.PresentationVoiceFeature) {
  return {
    title: v.title,
    description: v.description,
    capability_id: v.capabilityId,
  };
}

function voiceStory(v: Wire.PresentationVoiceStory): ContentByKind['voice-story'] {
  return {
    eyebrow: v.eyebrow,
    heading: v.heading,
    body: v.body,
    features: v.features.map(voiceFeature),
    note: v.note,
    input_label: v.inputLabel,
    transcript: v.transcript,
    summary_label: v.summaryLabel,
    summary_title: v.summaryTitle,
    summary_items: [...v.summaryItems],
    output_label: v.outputLabel,
    demo_note: v.demoNote,
    provider_qualification: v.providerQualification,
    waveform: [...v.waveform],
    capability_ids: [...v.capabilityIds],
  };
}

function deviceStory(v: Wire.PresentationDeviceStory): ContentByKind['device-story'] {
  return {
    heading: v.heading,
    description: v.description,
    device: v.device,
    visual_ref: v.visualRef,
    alt_text: v.altText,
  };
}

function roadmap(v: Wire.PresentationCapabilityRoadmap): ContentByKind['capability-roadmap'] {
  return {
    heading: v.heading,
    capability_ids: [...v.capabilityIds],
    status_label: v.statusLabel,
    description: v.description,
  };
}

function pricing(v: Wire.PresentationPricing): ContentByKind['pricing'] {
  return {
    heading: v.heading,
    description: v.description,
    plan_refs: [...v.planRefs],
    actions: v.actions.map(action),
  };
}

function closingAction(v: Wire.PresentationClosingAction): ContentByKind['closing-action'] {
  return {
    heading: v.heading,
    description: v.description,
    actions: v.actions.map(action),
    visual_ref: v.visualRef,
  };
}

function faqItem(v: Wire.PresentationFAQItem) {
  return {
    question: v.question,
    answer: v.answer,
    accessible_label: v.accessibleLabel,
  };
}

function faq(v: Wire.PresentationFAQ): ContentByKind['faq'] {
  return {
    heading: v.heading,
    items: v.items.map(faqItem),
  };
}

function workspace(v: Wire.PresentationWorkspaceFixture): CanonicalWorkspace {
  return {
    title: v.title,
    group: v.group,
    group_label: v.groupLabel,
    groups: [...v.groups],
    sessions_label: v.sessionsLabel,
    sessions: [...v.sessions],
    role: v.role,
    model: v.model,
    reviewer: v.reviewer,
    reviewer_model: v.reviewerModel,
    branch: v.branch,
    prompt: v.prompt,
    answer: v.answer,
    files: [...v.files],
    file_label: v.fileLabel,
    diff: [...v.diff],
    command: v.command,
    checks: [...v.checks],
    ready: v.ready,
    composer: v.composer,
    return_label: v.returnLabel,
    return_title: v.returnTitle,
    review_message: v.reviewMessage,
    message_label: v.messageLabel,
    reply_label: v.replyLabel,
    status: v.status,
    today: v.today,
    keyboard: [...v.keyboard],
  };
}

function backdrop(v: Wire.PresentationBackdropFixture): CanonicalBackdrop {
  return {
    title: v.title,
    label: v.label,
    selected: v.selected,
    styles: [...v.styles],
    surface: v.surface,
    palette: v.palette,
    panel: v.panel,
    caption: v.caption,
    export: v.export,
    options: [...v.options],
    badge: v.badge,
    asset_refs: [...v.assetRefs],
  };
}

function workflowStep(v: Wire.PresentationWorkflowStep) {
  return {
    number: v.number,
    title: v.title,
    description: v.description,
  };
}

function workflow(v: Wire.PresentationWorkflowFixture): CanonicalWorkflow {
  return {
    title: v.title,
    label: v.label,
    steps: v.steps.map(workflowStep),
    browser_title: v.browserTitle,
    browser_rows: [...v.browserRows],
    note: v.note,
  };
}

function assetVariant(v: Wire.PresentationAssetVariant) {
  return {
    surface: v.surface,
    asset_id: v.assetId,
  };
}

function focalPoint(v: Wire.PresentationFocalPoint) {
  return {
    x: v.x,
    y: v.y,
  };
}

function provenance(v: Wire.PresentationAssetProvenance) {
  return {
    provider: v.provider,
    job_ref: v.jobRef,
    candidate_ref: v.candidateRef,
  };
}

function legibility(v: Wire.ResolvedPresentationLegibility) {
  return {
    contrast_ratio: v.contrastRatio,
    minimum_contrast_ratio: v.minimumContrastRatio,
    threshold: v.threshold,
    verdict: oneOf<'pass' | 'fail' | 'not_measured'>(v.verdict, ['pass', 'fail', 'not_measured']),
  };
}

function overlayRegion(v: Wire.ResolvedPresentationOverlayRegion) {
  return {
    name: v.name,
    x: v.x,
    y: v.y,
    width: v.width,
    height: v.height,
    measurement: legibility(required(v.measurement, 'measurement')),
  };
}

function asset(v: Wire.ResolvedPresentationAsset): ResolvedAsset {
  return {
    id: v.id,
    release_ref: v.releaseRef,
    content_hash: v.contentHash,
    width: v.width,
    height: v.height,
    mime: v.mime,
    surface: v.surface,
    responsive_alternatives: v.responsiveAlternatives.map(assetVariant),
    focal_point: focalPoint(required(v.focalPoint, 'focal_point')),
    crop_policy: v.cropPolicy,
    provenance: provenance(required(v.provenance, 'provenance')),
    overlay_regions: v.overlayRegions.map(overlayRegion),
    public_url: v.publicUrl,
  };
}

function spotlight(v: Wire.PresentationAppSpotlight): Spotlight {
  return {
    app_key: v.appKey,
    slug: v.slug,
    name: v.name,
    tagline: v.tagline,
    description: v.description,
    detail_route: v.detailRoute,
  };
}

function capability(v: Wire.ResolvedPresentationCapability): Capability {
  return {
    id: v.id,
    label: v.label,
    benefits: [...v.benefits],
    status: oneOf<Capability['status']>(v.status, ['available', 'preview', 'coming-soon']),
    status_label: v.statusLabel,
    constraints: [...v.constraints],
    provider_requirements: [...v.providerRequirements],
    platform_requirements: [...v.platformRequirements],
  };
}

function display(v: Wire.PresentationPageDisplay): PresentationDisplay {
  const s = required(v.shell, 'page.display.shell');
  return {
    shell: {
      brand_name: s.brandName, brand_mark: mark(s.brandMark), brand_target: s.brandTarget,
      brand_subtitle: s.brandSubtitle, skip_label: s.skipLabel, menu_label: s.menuLabel,
      footer_brand_name: s.footerBrandName, footer_brand_mark: mark(s.footerBrandMark),
      footer_brand_target: s.footerBrandTarget, footer_tagline: s.footerTagline,
      copyright: s.copyright, footer_note: s.footerNote, unavailable_reason: s.unavailableReason,
      preview_label: s.previewLabel, header_action: s.headerAction ? action(s.headerAction) : undefined,
    },
    asset_labels: mapValues(v.assetLabels, label => ({ alt: label.alt, sizes: label.sizes || undefined })),
    fixture_display: mapValues(v.fixtureDisplay, f => ({
      mark: mark(f.mark), avatar: f.avatar, time: f.time, tabs_label: f.tabsLabel,
      terminal_label: f.terminalLabel, messages_label: f.messagesLabel, file_changes: { ...f.fileChanges },
    })),
    blocks: mapValues(v.blocks, b => ({
      eyebrow: b.eyebrow, description: b.description, note: b.note, accessibility_label: b.accessibilityLabel,
      badge: b.badge, mark: b.mark ? mark(b.mark) : undefined, formats: [...b.formats],
      anchors: { ...b.anchors }, heading_breaks: [...b.headingBreaks], fixture_ref: b.fixtureRef || undefined,
      hero_fixture_refs: { ...b.heroFixtureRefs },
    })),
    apps: mapValues(v.apps, a => ({
      fixture_ref: a.fixtureRef || undefined, visual_ref: a.visualRef || undefined, mark: mark(a.mark),
      tone: oneOf<'amber' | 'sage'>(a.tone, ['amber', 'sage']), detail_label: a.detailLabel,
    })),
  };
}

function fixture(v: Wire.PresentationFixture): ConfiguredFixture {
  if (v.kind !== v.data.case) throw new Error('Fixture kind and oneof disagree');
  switch (v.data.case) {
    case 'workspace': return { id: v.id, kind: v.data.case, workspace: workspace(v.data.value) };
    case 'backdrop': return { id: v.id, kind: v.data.case, backdrop: backdrop(v.data.value) };
    case 'workflow': return { id: v.id, kind: v.data.case, workflow: workflow(v.data.value) };
    default: return unreachable(v.data);
  }
}

function block(v: Wire.PresentationBlock): Block {
  const content = required(v.content, 'block.content').value;
  const base = { id: v.id, version: v.version, variant: v.variant };
  let result: Block;
  switch (content.case) {
    case 'productHero': result = { ...base, kind: 'product-hero', content: productHero(content.value) }; break;
    case 'bundleHero': result = { ...base, kind: 'bundle-hero', content: bundleHero(content.value) }; break;
    case 'capabilityStrip': result = { ...base, kind: 'capability-strip', content: capabilityStrip(content.value) }; break;
    case 'productStory': result = { ...base, kind: 'product-story', content: productStory(content.value) }; break;
    case 'productDemo': result = { ...base, kind: 'product-demo', content: productDemo(content.value) }; break;
    case 'appSpotlights': result = { ...base, kind: 'app-spotlights', content: appSpotlights(content.value) }; break;
    case 'artifactExplorer': result = { ...base, kind: 'artifact-explorer', content: artifactExplorer(content.value) }; break;
    case 'voiceStory': result = { ...base, kind: 'voice-story', content: voiceStory(content.value) }; break;
    case 'deviceStory': result = { ...base, kind: 'device-story', content: deviceStory(content.value) }; break;
    case 'capabilityRoadmap': result = { ...base, kind: 'capability-roadmap', content: roadmap(content.value) }; break;
    case 'pricing': result = { ...base, kind: 'pricing', content: pricing(content.value) }; break;
    case 'closingAction': result = { ...base, kind: 'closing-action', content: closingAction(content.value) }; break;
    case 'faq': result = { ...base, kind: 'faq', content: faq(content.value) }; break;
    case 'footer': result = { ...base, kind: 'footer', content: footer(content.value) }; break;
    case undefined: throw new Error('Missing block content');
    default: return unreachable(content);
  }
  if (v.kind !== result.kind) throw new Error('Block kind and oneof disagree');
  return result;
}

/** One fail-closed boundary for generated public config and authorized private preview. */
export function decodeProductPresentation(value: Wire.ResolvedProductPresentation): Presentation {
  rejectUnknown(value);
  const p = required(value.page, 'page');
  const theme = required(p.theme, 'page.theme');
  const nav = required(p.navigation, 'page.navigation');
  const diagnostics = required(value.diagnostics, 'diagnostics');
  const result: Presentation = {
    schema_version: value.schemaVersion,
    mode: oneOf<Presentation['mode']>(value.mode, ['empty', 'single_app', 'bundle', 'app_detail']),
    scope: oneOf<Presentation['scope']>(value.scope, ['app', 'bundle']), app_key: value.appKey,
    selected_app_keys: [...value.selectedAppKeys], spotlights: value.spotlights.map(spotlight),
    capabilities: value.capabilities.map(capability), fixtures: value.fixtures.map(fixture), assets: value.assets.map(asset),
    page: {
      id: p.id, locale: p.locale, title: p.title, description: p.description,
      theme: { variant: oneOf<'signal' | 'studio'>(theme.variant, ['signal', 'studio']), primary: theme.primary, background: theme.background, accent: theme.accent },
      navigation: { label: nav.label, items: nav.items.map(navigationItem) },
      blocks: p.blocks.map(block), footer: footer(required(p.footer, 'page.footer')),
      display: display(required(p.display, 'page.display')),
    },
    diagnostics: {
      preview: diagnostics.preview, eligible_app_keys: [...diagnostics.eligibleAppKeys],
      noindex: diagnostics.noindex, no_store: diagnostics.noStore, resolved_revision: diagnostics.resolvedRevision, fallback: diagnostics.fallback,
    },
  };
  if (diagnostics.preview && (!diagnostics.noindex || !diagnostics.noStore)) throw new Error('Unsafe preview diagnostics');
  assertRendererContract(result);
  resolveResources(result);
  return result;
}

/** Dev/tests or explicit protobuf JSON only; production callers pass generated messages. */
export function parseProductPresentation(text: string): Presentation {
  return decodeProductPresentation(fromJsonString(ResolvedProductPresentationSchema, text, { ignoreUnknownFields: false }));
}
