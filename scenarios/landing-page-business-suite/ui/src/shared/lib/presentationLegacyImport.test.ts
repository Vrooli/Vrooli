import { create } from '@bufbuild/protobuf';
import { readFileSync, writeFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import {
  ProductPresentationDocumentSchema,
  PresentationBlockContentSchema,
  PresentationBlockSchema,
  PresentationAppSchema,
  PresentationStringTableSchema,
  PresentationShellDisplaySchema,
} from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';
import { formatPresentationDocument, parsePresentationDocument } from '../api/productPresentation';
import { importLegacyPresentationSnapshot } from './presentationLegacyImport';

function targetDocument(appKey = 'web-console', locale = 'en') {
  return create(ProductPresentationDocumentSchema, {
    schemaVersion: 1,
    bundle: { key: 'business_suite', name: 'Configured suite', pageId: 'bundle-page', emptyPageId: 'bundle-page', defaultLocale: locale, locales: [locale] },
    apps: [{ key: appKey, slug: 'aquila', name: 'Aquila', enabled: false, visibility: 'private', publication: 'draft', pageId: 'aquila-page', preservationRef: 'bas-preservation-ref' }],
    pages: [{
      id: 'aquila-page', locale, title: 'Existing private draft', description: 'Existing description',
      display: { shell: create(PresentationShellDisplaySchema, { unavailableReason: 'Configured unavailable reason' }) },
      blocks: [create(PresentationBlockSchema, { id: 'existing', kind: 'faq', version: 1, variant: 'accordion', content: create(PresentationBlockContentSchema, { value: { case: 'faq', value: { heading: 'Existing copy', items: [] } } }) })],
    }],
  });
}

const legacySource = JSON.stringify({
  variant: { slug: 'control', name: 'Legacy control', old_file: 'private.json' },
  sections: [
    { key: 'hero', section_type: 'hero', order: 2, content: { title: 'Legacy <strong>title</strong>', subtitle: 'Legacy intro' } },
    { key: 'features', section_type: 'features', order: 1, content: { title: 'Legacy features', subtitle: 'Legacy feature intro', features: [{ title: 'Durable work', description: 'Keeps context.' }] } },
    { key: 'faq', section_type: 'faq', order: 3, content: { title: 'Questions', items: [{ question: 'Can I import?', answer: 'Yes.' }] } },
    { key: 'download', section_type: 'download', order: 4, content: { cta_text: 'Download now', cta_url: 'https://private.example/download' } },
    { key: 'pricing', section_type: 'pricing', order: 5, content: { title: 'Never activate this price', price: '$99' } },
  ],
});

function sourceFromReceipt(document: ReturnType<typeof targetDocument>, keys: string[]): string {
  const values = document.strings.en?.values ?? {};
  const encoded = keys.map((key) => values[key] ?? '').join('');
  const binary = atob(encoded);
  return new TextDecoder().decode(Uint8Array.from(binary, (character) => character.charCodeAt(0)));
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function unescapePointerToken(value: string): string {
  return value.replace(/~1/g, '/').replace(/~0/g, '~');
}

function resolvePointer(root: unknown, pointer: string): { found: boolean; value: unknown } {
  if (pointer === '') return { found: true, value: root };
  let current: unknown = root;
  for (const encodedToken of pointer.slice(1).split('/')) {
    const token = unescapePointerToken(encodedToken);
    if (Array.isArray(current)) {
      const index = Number(token);
      if (!Number.isInteger(index) || index < 0 || index >= current.length) return { found: false, value: undefined };
      current = current[index];
    } else if (isObject(current) && token in current) {
      current = current[token];
    } else {
      return { found: false, value: undefined };
    }
  }
  return { found: true, value: current };
}

function parseUnknown(text: string): unknown {
  return JSON.parse(text);
}

function normalizeForTest(value: string): string {
  return value.split(String.fromCharCode(0)).join('').replace(/[\r\n\t]+/g, ' ').replace(/\s+/g, ' ').trim();
}

function documentWithValues(document: ReturnType<typeof targetDocument>, values: Record<string, string>) {
  const copy = parsePresentationDocument(formatPresentationDocument(document));
  copy.strings.en = create(PresentationStringTableSchema, { values });
  return copy;
}

function namespaceFromReceipt(receipt: { sourceOriginal: string[] }): string {
  const first = receipt.sourceOriginal[0];
  if (!first) throw new Error('missing source chunk');
  const marker = '-source-';
  const index = first.lastIndexOf(marker);
  if (index < 0) throw new Error('invalid source chunk name');
  return first.slice(0, index);
}

function encodeBase64(bytes: Uint8Array): string {
  let binary = '';
  bytes.forEach((byte) => { binary += String.fromCharCode(byte); });
  return btoa(binary);
}

function decodeBase64(value: string): Uint8Array {
  const binary = atob(value);
  return Uint8Array.from(binary, (character) => character.charCodeAt(0));
}

function encodedChunks(bytes: Uint8Array, prefix: string): Record<string, string> {
  const values: Record<string, string> = {};
  const chunkSize = 3072;
  const count = Math.max(1, Math.ceil(bytes.byteLength / chunkSize));
  for (let index = 0; index < count; index += 1) {
    values[`${prefix}-${String(index).padStart(4, '0')}`] = encodeBase64(bytes.slice(index * chunkSize, (index + 1) * chunkSize));
  }
  return values;
}

function receiptValuesWithTargetEdit(document: ReturnType<typeof targetDocument>, receipt: { sourceOriginal: string[] }): Record<string, string> {
  const values = document.strings.en?.values ?? {};
  const namespace = namespaceFromReceipt(receipt);
  const receiptPrefix = `${namespace}-receipt-`;
  const receiptKeys = Object.keys(values).filter((key) => key.startsWith(receiptPrefix)).sort();
  const receiptText = receiptKeys.map((key) => values[key] ?? '').map(decodeBase64).reduce((all, bytes) => {
    const merged = new Uint8Array(all.byteLength + bytes.byteLength);
    merged.set(all);
    merged.set(bytes, all.byteLength);
    return merged;
  }, new Uint8Array());
  const parsed = parseUnknown(new TextDecoder().decode(receiptText));
  if (!isObject(parsed) || !isObject(parsed.target)) throw new Error('invalid test receipt');
  parsed.target.pageId = 'edited-page-id';
  const withoutReceipt = Object.fromEntries(Object.entries(values).filter(([key]) => !key.startsWith(receiptPrefix)));
  return { ...withoutReceipt, ...encodedChunks(new TextEncoder().encode(JSON.stringify(parsed)), `${namespace}-receipt`) };
}

describe('local legacy presentation adapter v1', () => {
  it('maps supported narrative sections in numeric order and retains exact source bytes privately', async () => {
    const original = targetDocument();
    const before = formatPresentationDocument(original);
    const result = await importLegacyPresentationSnapshot(original, legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' });
    expect(formatPresentationDocument(original)).toBe(before);
    expect(result.document.pages[0]?.blocks.map((block) => `${block.kind}:${block.variant}`)).toEqual([
      'faq:accordion', 'product-story:three-column', 'product-story:three-column', 'faq:accordion', 'closing-action:plain',
    ]);
    const page = result.document.pages[0];
    expect(page?.blocks[1]?.content?.value.case).toBe('productStory');
    expect(page?.blocks[4]?.content?.value.case).toBe('closingAction');
    if (page?.blocks[4]?.content?.value.case === 'closingAction') {
      expect(page.blocks[4].content.value.value.actions[0]?.kind).toBe('unavailable');
      expect(page.blocks[4].content.value.value.actions[0]?.target).toBe('');
      expect(page.blocks[4].content.value.value.actions[0]?.reason).toBe('Configured unavailable reason');
    }
    expect(result.document.apps[0]?.preservationRef).toBe('bas-preservation-ref');
    expect(sourceFromReceipt(result.document, result.receipt.sourceOriginal)).toBe(legacySource);
    expect(result.receipt.sha256).toMatch(/^sha256:[a-f0-9]{64}$/);
    expect(result.receipt.fieldLevelDispositions.some((item) => item.sourcePath.endsWith('cta_url') && item.disposition === 'preserved-only')).toBe(true);
    expect(result.receipt.fieldLevelDispositions.some((item) => item.sourcePath.endsWith('title') && item.disposition !== 'preserved-only')).toBe(true);
    expect(result.document.pages[0]?.blocks[1]?.content?.value.case === 'productStory' && JSON.stringify(result.document.pages[0].blocks[1].content.value.value).includes('https://')).toBe(false);

    const sourceWire = parseUnknown(legacySource);
    const documentWire = parseUnknown(formatPresentationDocument(result.document));
    for (const disposition of result.receipt.fieldLevelDispositions) {
      expect(resolvePointer(sourceWire, disposition.sourcePath).found, disposition.sourcePath).toBe(true);
      if (disposition.disposition === 'mapped' || disposition.disposition === 'normalized') {
        expect(disposition.targetPaths?.length).toBeGreaterThan(0);
        for (const targetPath of disposition.targetPaths ?? []) {
          const targetValue = resolvePointer(documentWire, targetPath);
          expect(targetValue.found, targetPath).toBe(true);
          const sourceValue = resolvePointer(sourceWire, disposition.sourcePath).value;
          if (typeof sourceValue === 'string') expect(targetValue.value).toBe(normalizeForTest(sourceValue));
        }
      }
    }
    expect(result.receipt.fieldLevelDispositions.some((item) => item.sourcePath.includes('cta_url'))).toBe(true);
    expect(result.receipt.fieldLevelDispositions.some((item) => item.sourcePath.includes('sections/1/content/features/0'))).toBe(true);
  });

  it('is idempotent for the same source and target and refuses occupied namespaces', async () => {
    const first = await importLegacyPresentationSnapshot(targetDocument(), legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' });
    const second = await importLegacyPresentationSnapshot(first.document, legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' });
    expect(formatPresentationDocument(second.document)).toBe(formatPresentationDocument(first.document));
    expect(second.receipt).toEqual(first.receipt);
    const occupied = parsePresentationDocument(formatPresentationDocument(targetDocument()));
    const occupiedKey = first.receipt.sourceOriginal[0];
    if (!occupiedKey) throw new Error('missing source key');
    occupied.strings.en = create(PresentationStringTableSchema, { values: { [occupiedKey]: 'occupied' } });
    await expect(importLegacyPresentationSnapshot(occupied, legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow('collision');
  });

  it('keeps the namespace target-specific for the same source hash', async () => {
    const en = await importLegacyPresentationSnapshot(targetDocument('web-console', 'en'), legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' });
    const fr = await importLegacyPresentationSnapshot(targetDocument('other-app', 'fr'), legacySource, { adapterVersion: 1, appKey: 'other-app', locale: 'fr' });
    expect(en.receipt.sha256).toBe(fr.receipt.sha256);
    expect(en.receipt.target).not.toEqual(fr.receipt.target);
    expect(en.receipt.sourceOriginal[0]).not.toBe(fr.receipt.sourceOriginal[0]);
  });

  it.each([
    ['unsupported version', legacySource, { adapterVersion: 2, appKey: 'web-console', locale: 'en' }],
    ['missing app key', legacySource, { adapterVersion: 1, appKey: '', locale: 'en' }],
    ['public target', legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' }],
  ])('%s is rejected without mutating input', async (label, source, options) => {
    const document = targetDocument();
    if (label === 'public target') {
      const app = document.apps[0];
      if (!app) throw new Error('missing test app');
      app.visibility = 'public';
    }
    const before = formatPresentationDocument(document);
    await expect(importLegacyPresentationSnapshot(document, source, options)).rejects.toThrow();
    expect(formatPresentationDocument(document)).toBe(before);
  });

  it.each([
    ['malformed', '{"variant":', 'malformed'],
    ['duplicate keys', '{"variant":{},"sections":[],"sections":[]}', 'duplicate JSON key'],
    ['duplicate section key', '{"variant":{},"sections":[{"key":"same","section_type":"hero","content":{}},{"key":"same","section_type":"faq","content":{}}]}', 'duplicate section key'],
    ['duplicate section identity', '{"variant":{},"sections":[{"section_type":"hero","order":1,"content":{}},{"section_type":"hero","order":1,"content":{}}]}', 'duplicate section identity'],
  ])('%s is rejected', async (_label, source, reason) => {
    await expect(importLegacyPresentationSnapshot(targetDocument(), source, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow(reason);
  });

  it('rejects an oversized source before changing the document', async () => {
    const source = JSON.stringify({ variant: {}, sections: [{ section_type: 'hero', content: { title: 'x'.repeat(1024 * 1024) } }] });
    const document = targetDocument();
    const before = formatPresentationDocument(document);
    await expect(importLegacyPresentationSnapshot(document, source, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow('1 MiB');
    expect(formatPresentationDocument(document)).toBe(before);
  });

  it('uses collection aliases and maps the first valid FAQ item, not an invalid first item', async () => {
    const source = JSON.stringify({
      variant: { slug: 'aliases' },
      sections: [
        {
          section_type: 'features',
          content: { items: [{ title: 'incomplete', description: '' }, { title: 'Feature title', description: 'Feature description' }] },
        },
        {
          section_type: 'faq',
          content: { faqs: [{ question: 'missing answer' }, { title: 'Valid question', answer: 'Valid answer', accessible_label: '   ' }] },
        },
      ],
    });
    const result = await importLegacyPresentationSnapshot(targetDocument(), source, { adapterVersion: 1, appKey: 'web-console', locale: 'en' });
    const mapped = result.receipt.fieldLevelDispositions.filter((item) => item.disposition !== 'preserved-only');
    expect(mapped.some((item) => item.sourcePath === '/sections/0/content/items/0/title')).toBe(false);
    expect(mapped.some((item) => item.sourcePath === '/sections/0/content/items/1/title')).toBe(true);
    expect(mapped.some((item) => item.sourcePath === '/sections/1/content/faqs/0/question')).toBe(false);
    expect(mapped.some((item) => item.sourcePath === '/sections/1/content/faqs/1/title')).toBe(true);
    expect(mapped.some((item) => item.sourcePath === '/sections/1/content/faqs/1/title' && item.targetPaths?.some((path) => path.endsWith('/heading')))).toBe(true);
    expect(mapped.some((item) => item.sourcePath === '/sections/0/content/items/1/title' && item.targetPaths?.some((path) => path.endsWith('/product_story/heading')))).toBe(true);
    expect(mapped.some((item) => item.sourcePath === '/sections/0/content/items/1/description' && item.targetPaths?.some((path) => path.endsWith('/product_story/body')))).toBe(true);
    expect(mapped.some((item) => item.sourcePath === '/sections/1/content/faqs/1/title' && item.targetPaths?.some((path) => path.endsWith('/accessible_label')))).toBe(true);
    for (const item of mapped) {
      for (const path of item.targetPaths ?? []) expect(path).toMatch(/^\/pages\/0\/blocks\/\d+\/content\/(product_story|faq)\//);
    }
  });

  it('maps title-only recovery using canonical target pointers and preserves dotted source keys', async () => {
    const source = JSON.stringify({
      variant: { 'odd.key/slash~': 'kept' },
      sections: [{ key: 'odd.key/slash~', section_type: 'hero', content: { title: '  Unicode café  ', 'unknown.key/slash~': 'preserve me' } }],
      'unknown.field/name': 'preserve me',
    });
    const result = await importLegacyPresentationSnapshot(targetDocument(), source, { adapterVersion: 1, appKey: 'web-console', locale: 'en' });
    const sourceWire = parseUnknown(source);
    const documentWire = parseUnknown(formatPresentationDocument(result.document));
    const titleDisposition = result.receipt.fieldLevelDispositions.find((item) => item.sourcePath === '/sections/0/content/title');
    expect(titleDisposition?.disposition).toBe('normalized');
    expect(titleDisposition?.targetPaths).toEqual([
      '/pages/0/blocks/1/content/product_story/heading',
      '/pages/0/blocks/1/content/product_story/body',
      '/pages/0/blocks/1/content/product_story/items/0/title',
      '/pages/0/blocks/1/content/product_story/items/0/description',
    ]);
    expect(result.receipt.fieldLevelDispositions.some((item) => item.sourcePath === '/sections/0/content/unknown.key~1slash~0')).toBe(true);
    expect(resolvePointer(sourceWire, '/sections/0/content/unknown.key~1slash~0').found).toBe(true);
    expect(resolvePointer(sourceWire, '/unknown.field~1name').found).toBe(true);
    for (const disposition of result.receipt.fieldLevelDispositions) {
      expect(resolvePointer(sourceWire, disposition.sourcePath).found).toBe(true);
      for (const path of disposition.targetPaths ?? []) expect(resolvePointer(documentWire, path).found).toBe(true);
    }
  });

  it('refuses a corrupted, incomplete, extra, or edited stored provenance ledger', async () => {
    const first = await importLegacyPresentationSnapshot(targetDocument(), legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' });
    const values = first.document.strings.en?.values ?? {};
    const namespace = namespaceFromReceipt(first.receipt);
    const sourceKey = first.receipt.sourceOriginal[0];
    if (!sourceKey) throw new Error('missing source key');
    const sourceCorrupt = { ...values, [sourceKey]: '%%%%' };
    await expect(importLegacyPresentationSnapshot(documentWithValues(first.document, sourceCorrupt), legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow(/stored (base64|source)/);

    const missing = Object.fromEntries(Object.entries(values).filter(([key]) => key !== sourceKey));
    await expect(importLegacyPresentationSnapshot(documentWithValues(first.document, missing), legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow(/storage namespace collision|stored (chunk|source)/);

    const extra = { ...values, [`${namespace}-source-9999`]: '' };
    await expect(importLegacyPresentationSnapshot(documentWithValues(first.document, extra), legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow('collision');

    const editedReceiptKey = `${namespace}-receipt-0000`;
    const editedReceipt = { ...values, [editedReceiptKey]: btoa('{}') };
    await expect(importLegacyPresentationSnapshot(documentWithValues(first.document, editedReceipt), legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow('collision');
  });

  it('refuses excessive narrative size, nesting, and bundle-page aliases without changing the input', async () => {
    const largeSource = JSON.stringify({ variant: {}, sections: [{ section_type: 'hero', content: { title: 'x'.repeat(10001) } }] });
    const largeResult = await importLegacyPresentationSnapshot(targetDocument(), largeSource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' });
    expect(largeResult.document.pages[0]?.blocks).toHaveLength(1);
    expect(largeResult.receipt.fieldLevelDispositions.find((item) => item.sourcePath === '/sections/0/content/title')?.disposition).toBe('preserved-only');

    let nested: Record<string, unknown> = { leaf: 'value' };
    for (let index = 0; index < 130; index += 1) nested = { nested };
    const deepSource = JSON.stringify({ variant: {}, sections: [], oldfiles: nested });
    await expect(importLegacyPresentationSnapshot(targetDocument(), deepSource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow('nesting');

    const aliased = targetDocument();
    const app = aliased.apps[0];
    if (!app) throw new Error('missing test app');
    const bundlePageId = aliased.bundle?.pageId;
    if (!bundlePageId) throw new Error('missing bundle page');
    app.pageId = bundlePageId;
    await expect(importLegacyPresentationSnapshot(aliased, legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow('alias');
  });

  it('refuses valid-base64 source edits and missing or out-of-order receipt chunks', async () => {
    const first = await importLegacyPresentationSnapshot(targetDocument(), legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' });
    const values = first.document.strings.en?.values ?? {};
    const sourceKey = first.receipt.sourceOriginal[0];
    if (!sourceKey) throw new Error('missing source key');
    const sourceBytes = decodeBase64(values[sourceKey] ?? '');
    sourceBytes[0] = (sourceBytes[0] ?? 0) ^ 1;
    await expect(importLegacyPresentationSnapshot(documentWithValues(first.document, { ...values, [sourceKey]: encodeBase64(sourceBytes) }), legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow('source bytes do not match');

    const namespace = namespaceFromReceipt(first.receipt);
    const receiptPrefix = `${namespace}-receipt-`;
    const receiptKeys = Object.keys(values).filter((key) => key.startsWith(receiptPrefix)).sort();
    if (receiptKeys.length < 2) throw new Error('test receipt must span multiple chunks');
    const missing = Object.fromEntries(Object.entries(values).filter(([key]) => key !== receiptKeys[receiptKeys.length - 1]));
    await expect(importLegacyPresentationSnapshot(documentWithValues(first.document, missing), legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow(/receipt|collision/);
    const firstReceiptKey = receiptKeys[0];
    if (!firstReceiptKey) throw new Error('missing first receipt key');
    const outOfOrder = { ...Object.fromEntries(Object.entries(values).filter(([key]) => key !== firstReceiptKey)), [`${namespace}-receipt-9999`]: values[firstReceiptKey] ?? '' };
    await expect(importLegacyPresentationSnapshot(documentWithValues(first.document, outOfOrder), legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow(/receipt|collision/);
  });

  it.each([
    ['ambiguous app', (document: ReturnType<typeof targetDocument>) => { const app = document.apps[0]; if (!app) throw new Error('missing app'); document.apps.push(app); }],
    ['ambiguous page owner', (document: ReturnType<typeof targetDocument>) => { document.apps.push(create(PresentationAppSchema, { key: 'other-owner', slug: 'other-owner', enabled: false, visibility: 'private', publication: 'draft', pageId: 'aquila-page' })); }],
    ['ambiguous exact locale', (document: ReturnType<typeof targetDocument>) => { const page = document.pages[0]; if (!page) throw new Error('missing page'); document.pages.push(page); }],
  ])('refuses %s target identity', async (_label, mutate) => {
    const document = targetDocument();
    mutate(document);
    await expect(importLegacyPresentationSnapshot(document, legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow();
  });

  it('refuses a receipt edited with a valid target/page payload', async () => {
    const first = await importLegacyPresentationSnapshot(targetDocument(), legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' });
    const values = receiptValuesWithTargetEdit(first.document, first.receipt);
    await expect(importLegacyPresentationSnapshot(documentWithValues(first.document, values), legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow(/receipt|collision/);
  });

  it.each([
    ['missing variant', JSON.stringify({ sections: [] }), 'variant'],
    ['missing sections', JSON.stringify({ variant: {} }), 'sections'],
    ['invalid enabled', JSON.stringify({ variant: {}, sections: [{ section_type: 'hero', enabled: 'yes', content: {} }] }), 'enabled'],
    ['invalid order', JSON.stringify({ variant: {}, sections: [{ section_type: 'hero', order: 0, content: {} }] }), 'order'],
  ])('refuses unsupported envelope/section input: %s', async (_label, source, reason) => {
    await expect(importLegacyPresentationSnapshot(targetDocument(), source, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow(reason);
  });

  it('keeps an unsupported section preserved-only instead of emitting a recovery block', async () => {
    const source = JSON.stringify({ variant: {}, sections: [{ section_type: 'pricing', content: { title: 'Do not render this' } }] });
    const result = await importLegacyPresentationSnapshot(targetDocument(), source, { adapterVersion: 1, appKey: 'web-console', locale: 'en' });
    expect(result.document.pages[0]?.blocks).toHaveLength(1);
    expect(result.receipt.fieldLevelDispositions.find((item) => item.sourcePath === '/sections/0/content/title')?.disposition).toBe('preserved-only');
  });

  it('rejects an output that would exceed 8 MiB without mutating the input document', async () => {
    const document = targetDocument();
    const page = document.pages[0];
    if (!page) throw new Error('missing test page');
    page.description = 'x'.repeat(7_400_000);
    const before = formatPresentationDocument(document);
    expect(new TextEncoder().encode(before).byteLength).toBeLessThan(8 * 1024 * 1024);
    const source = JSON.stringify({ variant: {}, sections: [], oldfiles: 'y'.repeat(900_000) });
    await expect(importLegacyPresentationSnapshot(document, source, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow(/8 MiB/);
    expect(formatPresentationDocument(document)).toBe(before);
  });

  it('preserves escaped quotes and backslashes in the exact private source ledger', async () => {
    const source = JSON.stringify({
      variant: { slug: 'escaped', old_file: 'legacy\\quoted.json' },
      sections: [{ section_type: 'hero', content: { title: 'He said "go" \\ now', subtitle: 'line\\break' } }],
    });
    const result = await importLegacyPresentationSnapshot(targetDocument(), source, { adapterVersion: 1, appKey: 'web-console', locale: 'en' });
    expect(sourceFromReceipt(result.document, result.receipt.sourceOriginal)).toBe(source);
    expect(result.receipt.fieldLevelDispositions.some((item) => item.sourcePath === '/variant/old_file')).toBe(true);
  });

  it('refuses import when Web Crypto subtle is unavailable and restores the global', async () => {
    const descriptor = Object.getOwnPropertyDescriptor(globalThis, 'crypto');
    try {
      Object.defineProperty(globalThis, 'crypto', { configurable: true, value: { subtle: undefined } });
      await expect(importLegacyPresentationSnapshot(targetDocument(), legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow();
    } finally {
      if (descriptor) Object.defineProperty(globalThis, 'crypto', descriptor);
      else Reflect.deleteProperty(globalThis, 'crypto');
    }
    expect(globalThis.crypto.subtle).toBeDefined();
  });

  it('refuses an unpaired UTF-16 source before mutating the input document', async () => {
    const source = `{"variant":{},"sections":[{"section_type":"hero","content":{"title":"${String.fromCharCode(0xd800)}"}}]}`;
    const document = targetDocument();
    const before = formatPresentationDocument(document);
    await expect(importLegacyPresentationSnapshot(document, source, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow(/UTF-8/);
    expect(formatPresentationDocument(document)).toBe(before);
  });

  it('preserves a disabled supported hero as source-only without a recovery block', async () => {
    const source = JSON.stringify({ variant: {}, sections: [{ section_type: 'hero', enabled: false, content: { title: 'Disabled title', subtitle: 'Disabled body' } }] });
    const result = await importLegacyPresentationSnapshot(targetDocument(), source, { adapterVersion: 1, appKey: 'web-console', locale: 'en' });
    expect(result.document.pages[0]?.blocks).toHaveLength(1);
    expect(result.receipt.fieldLevelDispositions.find((item) => item.sourcePath === '/sections/0/content/title')?.disposition).toBe('preserved-only');
    expect(result.receipt.fieldLevelDispositions.some((item) => item.disposition !== 'preserved-only')).toBe(false);
  });

  it('refuses a target without a configured unavailable reason without mutating it', async () => {
    const document = targetDocument();
    const page = document.pages[0];
    if (!page?.display?.shell) throw new Error('missing test shell');
    page.display.shell.unavailableReason = '';
    const before = formatPresentationDocument(document);
    await expect(importLegacyPresentationSnapshot(document, legacySource, { adapterVersion: 1, appKey: 'web-console', locale: 'en' })).rejects.toThrow('unavailable reason');
    expect(formatPresentationDocument(document)).toBe(before);
  });

  it.skipIf(!process.env.LPBS_LEGACY_SEED_WIRE_INPUT || !process.env.LPBS_LEGACY_IMPORT_SOURCE || !process.env.LPBS_LEGACY_IMPORT_DOCUMENT)('imports an explicitly supplied canonical seed wire without default writes', async () => {
    const inputPath = process.env.LPBS_LEGACY_SEED_WIRE_INPUT;
    const sourcePath = process.env.LPBS_LEGACY_IMPORT_SOURCE;
    const outputPath = process.env.LPBS_LEGACY_IMPORT_DOCUMENT;
    if (!inputPath || !sourcePath || !outputPath) throw new Error('integration paths are incomplete');
    const input = parsePresentationDocument(readFileSync(inputPath, 'utf8'));
    const source = readFileSync(sourcePath, 'utf8');
    const result = await importLegacyPresentationSnapshot(input, source, { adapterVersion: 1, appKey: 'browser-automation-studio', locale: 'en' });
    writeFileSync(outputPath, formatPresentationDocument(result.document), { flag: 'wx' });
    const receiptPath = process.env.LPBS_LEGACY_IMPORT_RECEIPT;
    if (receiptPath) writeFileSync(receiptPath, JSON.stringify(result.receipt, null, 2), { flag: 'wx' });
  });
});
