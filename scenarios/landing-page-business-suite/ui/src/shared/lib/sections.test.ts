import { describe, expect, it } from 'vitest';
import { DOWNLOAD_ANCHOR_ID, getSectionAnchorId, getSectionKey } from './sections';

describe('landing section identifiers', () => {
  it('uses stable ids when present and deterministic type/order fallbacks when they are absent', () => {
    expect(getSectionKey({ id: 42, section_type: 'feature_grid', content: {}, order: 3 })).toBe(42);
    expect(getSectionKey({ section_type: 'feature_grid', content: {}, order: 3 })).toBe('feature_grid-3');
  });

  it('uses the download anchor and normalizes ordinary section anchors', () => {
    expect(getSectionAnchorId({ section_type: 'downloads', content: {}, order: 4 })).toBe(DOWNLOAD_ANCHOR_ID);
    expect(getSectionAnchorId({ id: 42, section_type: 'feature_grid', content: {}, order: 3 })).toBe('feature-grid-42');
    expect(getSectionAnchorId({ section_type: 'feature_grid', content: {}, order: 3 })).toBe('feature-grid-3');
  });
});
