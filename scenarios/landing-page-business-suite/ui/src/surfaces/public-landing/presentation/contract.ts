import type { Presentation, BlockKind } from './types';
/** Only implemented visual variants are accepted; unknown layouts never silently fall back. */
export const supportedVariants: Record<BlockKind, readonly string[]> = {
  'product-hero': ['centered'], 'bundle-hero': ['editorial'],
  'capability-strip': ['inline'], 'product-story': ['three-column'],
  'product-demo': ['static', 'interactive'], 'app-spotlights': ['grid'],
  'artifact-explorer': ['tabs'], 'voice-story': ['waveform', 'transcript'],
  'device-story': ['phone', 'desktop'], 'capability-roadmap': ['roadmap', 'stacked'],
  pricing: ['compact'], 'closing-action': ['plain', 'artwork'],
  faq: ['accordion', 'defined'], footer: ['defined', 'minimal'],
};
export function assertRendererContract(presentation: Presentation): void {
  if (presentation.schema_version !== 1) throw new Error('Unsupported presentation schema');
  const ids = new Set<string>();
  for (const block of presentation.page.blocks) {
    if (ids.has(block.id)) throw new Error('Duplicate presentation block id');
    ids.add(block.id);
    const variants = supportedVariants[block.kind];
    if (block.version !== 1 || !variants?.includes(block.variant)) {
      throw new Error(`Unsupported presentation block: ${block.kind}@${String(block.version)}/${block.variant}`);
    }
    if (block.kind === 'bundle-hero') {
      const keys = block.content.hero_items.map(item => item.app_key);
      if (new Set(keys).size !== keys.length) throw new Error('Duplicate hero app');
      if (keys.length > 3) throw new Error('Hero composition exceeds capacity');
      if (keys.some(key => !presentation.diagnostics.eligible_app_keys.includes(key) || !presentation.spotlights?.some(app => app.app_key === key))) throw new Error('Hero app is outside eligible resolved profiles');
      if (block.content.hero_items.some(item => !['artwork', 'screenshot', 'product-view', 'visual'].includes(item.exhibit_kind))) throw new Error('Unsupported hero exhibit');
    }
    if (block.kind === 'artifact-explorer') {
      const examples = block.content.examples;
      if (!examples.length || !examples.some(example => example.id === block.content.selected_example_id)) throw new Error('Invalid artifact selection');
      if (new Set(examples.map(example => example.id)).size !== examples.length) throw new Error('Duplicate artifact id');
    }
  }
}
