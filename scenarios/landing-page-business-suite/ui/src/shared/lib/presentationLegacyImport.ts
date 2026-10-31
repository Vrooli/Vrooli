import { create } from '@bufbuild/protobuf';
import {
  parsePresentationDocument,
  formatPresentationDocument,
} from '../api/productPresentation';
import type {
  PresentationBlock,
  PresentationFAQItem,
  PresentationProductStory,
  ProductPresentationDocument,
} from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';
import {
  PresentationActionSchema,
  PresentationBlockContentSchema,
  PresentationBlockSchema,
  PresentationClosingActionSchema,
  PresentationFAQItemSchema,
  PresentationFAQSchema,
  PresentationProductStorySchema,
  PresentationStoryItemSchema,
  PresentationStringTableSchema,
} from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';

const ADAPTER_VERSION = 1 as const;
const MAX_SOURCE_BYTES = 1024 * 1024;
const MAX_SECTIONS = 64;
const BASE64_CHUNK_BYTES = 3072;
const MAX_JSON_DEPTH = 128;
const MAX_NARRATIVE_RUNES = 10000;
const MAX_DOCUMENT_BYTES = 8 * 1024 * 1024;

type LegacyObject = Record<string, unknown>;
type Disposition = 'mapped' | 'normalized' | 'preserved-only';

export interface LegacyFieldDisposition {
  sourcePath: string;
  disposition: Disposition;
  targetPaths?: string[];
  normalization?: string;
  reason?: string;
}

export interface LegacyImportReceipt {
  receiptVersion: 1;
  adapterVersion: 1;
  sha256: string;
  sourceBytes: number;
  sourceOriginal: string[];
  target: {
    appKey: string;
    locale: string;
    pageId: string;
  };
  fieldLevelDispositions: LegacyFieldDisposition[];
}

export interface LegacyImportOptions {
  adapterVersion: number;
  appKey: string;
  locale: string;
}

export interface LegacyImportResult {
  document: ProductPresentationDocument;
  receipt: LegacyImportReceipt;
}

interface LegacySection {
  index: number;
  key?: string;
  sectionType: string;
  content: LegacyObject;
  order: number;
  enabled: boolean;
}

interface Mapping {
  targetPaths: string[];
  normalized: boolean;
}

interface TextSelection {
  key: string;
  value: string;
  normalized: boolean;
}

function mergeMappings(target: Map<string, Mapping>, source: Map<string, Mapping>): void {
  for (const [sourcePath, mapping] of source) {
    for (const targetPath of mapping.targetPaths) addMapping(target, sourcePath, targetPath, mapping.normalized);
  }
}

function isRecord(value: unknown): value is LegacyObject {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function cloneDocument(document: ProductPresentationDocument): ProductPresentationDocument {
  return parsePresentationDocument(formatPresentationDocument(document));
}

function error(message: string): Error {
  return new Error(`Legacy presentation import: ${message}`);
}

function scanStringEnd(text: string, start: number): number {
  let index = start + 1;
  while (index < text.length) {
    const character = text[index];
    if (character === '\\') {
      index += 2;
      continue;
    }
    if (character === '"') return index + 1;
    index += 1;
  }
  throw error('malformed JSON string');
}

function decodeJsonString(raw: string): string {
  let value: unknown;
  try {
    value = JSON.parse(raw);
  } catch {
    throw error('malformed JSON string');
  }
  if (typeof value !== 'string') throw error('malformed JSON key');
  return value;
}

/** JSON.parse keeps only the last duplicate key; scan first so imports cannot hide fields. */
function rejectDuplicateKeys(text: string): void {
  let index = 0;

  function skipWhitespace(): void {
    while (index < text.length && /\s/.test(text[index] ?? '')) index += 1;
  }

  function value(depth = 0): void {
    if (depth > MAX_JSON_DEPTH) throw error(`JSON nesting exceeds ${String(MAX_JSON_DEPTH)} levels`);
    skipWhitespace();
    const character = text[index];
    if (character === '{') {
      index += 1;
      skipWhitespace();
      const keys = new Set<string>();
      if (text[index] === '}') {
        index += 1;
        return;
      }
      while (index < text.length) {
        skipWhitespace();
        if (text[index] !== '"') throw error('malformed JSON object key');
        const end = scanStringEnd(text, index);
        const key = decodeJsonString(text.slice(index, end));
        if (keys.has(key)) throw error(`duplicate JSON key ${key}`);
        keys.add(key);
        index = end;
        skipWhitespace();
        if (text[index] !== ':') throw error('malformed JSON object');
        index += 1;
        value(depth + 1);
        skipWhitespace();
        if (text[index] === '}') {
          index += 1;
          return;
        }
        if (text[index] !== ',') throw error('malformed JSON object separator');
        index += 1;
      }
      throw error('unterminated JSON object');
    }
    if (character === '[') {
      index += 1;
      skipWhitespace();
      if (text[index] === ']') {
        index += 1;
        return;
      }
      while (index < text.length) {
        value(depth + 1);
        skipWhitespace();
        if (text[index] === ']') {
          index += 1;
          return;
        }
        if (text[index] !== ',') throw error('malformed JSON array separator');
        index += 1;
      }
      throw error('unterminated JSON array');
    }
    if (character === '"') {
      index = scanStringEnd(text, index);
      return;
    }
    const start = index;
    while (index < text.length && !/[\s,\]}]/.test(text[index] ?? '')) index += 1;
    if (start === index) throw error('malformed JSON value');
  }

  value();
  skipWhitespace();
  if (index !== text.length) throw error('trailing JSON data');
}

function normalizeText(value: string): { value: string; normalized: boolean } {
  const normalized = value
    .split(String.fromCharCode(0)).join('')
    .replace(/[\r\n\t]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
  return { value: normalized, normalized: normalized !== value };
}

function escapePointerToken(value: string): string {
  return value.replace(/~/g, '~0').replace(/\//g, '~1');
}

function sourcePointer(...tokens: Array<string | number>): string {
  return `/${tokens.map((token) => escapePointerToken(String(token))).join('/')}`;
}

function targetPointer(pageIndex: number, blockIndex: number, ...tokens: Array<string | number>): string {
  return sourcePointer('pages', pageIndex, 'blocks', blockIndex, ...tokens);
}

function addMapping(mappings: Map<string, Mapping>, sourcePath: string, targetPath: string, normalized: boolean): void {
  const current = mappings.get(sourcePath);
  if (current) {
    if (!current.targetPaths.includes(targetPath)) current.targetPaths.push(targetPath);
    current.normalized ||= normalized;
    return;
  }
  mappings.set(sourcePath, { targetPaths: [targetPath], normalized });
}

function textField(
  object: LegacyObject,
  key: string,
  sourcePath: string,
  targetPath: string,
  mappings: Map<string, Mapping>,
): string | undefined {
  const value = object[key];
  if (typeof value !== 'string') return undefined;
  const result = normalizeText(value);
  if (Array.from(result.value).length > MAX_NARRATIVE_RUNES) return undefined;
  if (!result.value) return undefined;
  addMapping(mappings, sourcePath, targetPath, result.normalized);
  return result.value;
}

function selectTextField(
  object: LegacyObject,
  keys: string[],
  sourcePrefix: string,
  targetPath: string,
  mappings: Map<string, Mapping>,
): TextSelection | undefined {
  for (const key of keys) {
    const candidate = object[key];
    if (typeof candidate !== 'string') continue;
    const result = normalizeText(candidate);
    if (!result.value || Array.from(result.value).length > MAX_NARRATIVE_RUNES) continue;
    addMapping(mappings, `${sourcePrefix}/${escapePointerToken(key)}`, targetPath, result.normalized);
    return { key, value: result.value, normalized: result.normalized };
  }
  return undefined;
}

function walkLeaves(value: unknown, path: string, leaves: string[]): void {
  if (Array.isArray(value)) {
    if (value.length === 0) leaves.push(path);
    value.forEach((child, index) => { walkLeaves(child, `${path}/${String(index)}`, leaves); });
    return;
  }
  if (isRecord(value)) {
    const keys = Object.keys(value).sort();
    if (keys.length === 0) leaves.push(path);
    keys.forEach((key) => { walkLeaves(value[key], `${path}/${escapePointerToken(key)}`, leaves); });
    return;
  }
  leaves.push(path);
}

function safeSegment(value: string, limit: number): string {
  const normalized = value.toLowerCase().replace(/[^a-z0-9._-]+/g, '-').replace(/^-+/, '');
  return (normalized.slice(0, limit) || 'target');
}

async function namespaceFor(appKey: string, locale: string, sourceHashHex: string): Promise<string> {
  const identity = new TextEncoder().encode(`${appKey}\u0000${locale}\u0000${sourceHashHex}`);
  const identityHash = await sha256(identity);
  return `legacy-v1-${safeSegment(appKey, 8)}-${safeSegment(locale, 5)}-${identityHash.slice(0, 24)}`;
}

function base64(bytes: Uint8Array): string {
  let binary = '';
  bytes.forEach((byte) => { binary += String.fromCharCode(byte); });
  return btoa(binary);
}

function chunks(bytes: Uint8Array, prefix: string): { keys: string[]; values: Record<string, string> } {
  const keys: string[] = [];
  const values: Record<string, string> = {};
  for (let offset = 0, chunkIndex = 0; offset < bytes.length; offset += BASE64_CHUNK_BYTES, chunkIndex += 1) {
    const key = `${prefix}-${String(chunkIndex).padStart(4, '0')}`;
    keys.push(key);
    values[key] = base64(bytes.slice(offset, offset + BASE64_CHUNK_BYTES));
  }
  if (bytes.length === 0) {
    const key = `${prefix}-0000`;
    keys.push(key);
    values[key] = '';
  }
  return { keys, values };
}

function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
}

async function sha256(bytes: Uint8Array): Promise<string> {
  const subtle = globalThis.crypto.subtle;
  return bytesToHex(new Uint8Array(await subtle.digest('SHA-256', bytes as BufferSource)));
}

function decodeBase64(value: string): Uint8Array {
  if (!/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(value)) {
    throw error('stored base64 is malformed');
  }
  let binary: string;
  try {
    binary = atob(value);
  } catch {
    throw error('stored base64 is malformed');
  }
  return Uint8Array.from(binary, (character) => character.charCodeAt(0));
}

function decodeStoredBytes(values: Record<string, string>, keys: string[]): Uint8Array {
  const bytes: number[] = [];
  for (const key of keys) {
    if (!(key in values)) throw error(`stored chunk ${key} is missing`);
    bytes.push(...decodeBase64(values[key] ?? ''));
  }
  return new Uint8Array(bytes);
}

function decodeStoredText(values: Record<string, string>, keys: string[]): string {
  try {
    return new TextDecoder('utf-8', { fatal: true }).decode(decodeStoredBytes(values, keys));
  } catch {
    throw error('stored text is not valid UTF-8');
  }
}

function expectedChunkKeys(prefix: string, byteLength: number): string[] {
  const count = Math.max(1, Math.ceil(byteLength / BASE64_CHUNK_BYTES));
  return Array.from({ length: count }, (_, index) => `${prefix}-${String(index).padStart(4, '0')}`);
}

function stringArray(value: unknown): string[] | undefined {
  if (!Array.isArray(value) || !value.every((item) => typeof item === 'string')) return undefined;
  return value;
}

function parseStoredReceipt(value: string): LegacyImportReceipt | undefined {
  let parsed: unknown;
  try {
    parsed = JSON.parse(value);
  } catch {
    return undefined;
  }
  if (!isRecord(parsed) || parsed.receiptVersion !== 1 || parsed.adapterVersion !== 1 || typeof parsed.sha256 !== 'string' || typeof parsed.sourceBytes !== 'number') return undefined;
  const sourceOriginal = stringArray(parsed.sourceOriginal);
  const target = parsed.target;
  if (!sourceOriginal || !isRecord(target) || typeof target.appKey !== 'string' || typeof target.locale !== 'string' || typeof target.pageId !== 'string') return undefined;
  if (!Array.isArray(parsed.fieldLevelDispositions)) return undefined;
  const fieldLevelDispositions: LegacyFieldDisposition[] = [];
  for (const item of parsed.fieldLevelDispositions) {
    if (!isRecord(item) || typeof item.sourcePath !== 'string' || (item.disposition !== 'mapped' && item.disposition !== 'normalized' && item.disposition !== 'preserved-only')) return undefined;
    const targetPaths = item.targetPaths === undefined ? undefined : stringArray(item.targetPaths);
    if (item.targetPaths !== undefined && !targetPaths) return undefined;
    fieldLevelDispositions.push({
      sourcePath: item.sourcePath,
      disposition: item.disposition,
      targetPaths,
      normalization: typeof item.normalization === 'string' ? item.normalization : undefined,
      reason: typeof item.reason === 'string' ? item.reason : undefined,
    });
  }
  return {
    receiptVersion: 1,
    adapterVersion: 1,
    sha256: parsed.sha256,
    sourceBytes: parsed.sourceBytes,
    sourceOriginal,
    target: { appKey: target.appKey, locale: target.locale, pageId: target.pageId },
    fieldLevelDispositions,
  };
}

function existingImport(
  values: Record<string, string>,
  namespace: string,
  sourceBytes: Uint8Array,
  sourceHash: string,
  appKey: string,
  locale: string,
  pageId: string,
  expectedReceiptText?: string,
): LegacyImportReceipt | undefined {
  const prefix = `${namespace}-`;
  const keys = Object.keys(values).filter((key) => key.startsWith(prefix));
  if (keys.length === 0) return undefined;
  const sourcePrefix = `${namespace}-source`;
  const receiptPrefix = `${namespace}-receipt`;
  const sourceKeys = keys.filter((key) => key.startsWith(`${sourcePrefix}-`)).sort();
  const receiptKeys = keys.filter((key) => key.startsWith(`${receiptPrefix}-`)).sort();
  const expectedSources = expectedChunkKeys(sourcePrefix, sourceBytes.byteLength);
  const expectedReceipts = expectedReceiptText
    ? expectedChunkKeys(receiptPrefix, new TextEncoder().encode(expectedReceiptText).byteLength)
    : expectedChunkKeys(receiptPrefix, decodeStoredBytes(values, receiptKeys).byteLength);
  const knownKeys = new Set([...expectedSources, ...expectedReceipts]);
  if (keys.some((key) => !knownKeys.has(key)) || sourceKeys.length === 0 || receiptKeys.length === 0) {
    throw error(`storage namespace collision for ${namespace}`);
  }
  if (sourceKeys.length !== expectedSources.length || sourceKeys.some((key, index) => key !== expectedSources[index])) {
    throw error(`stored source chunk sequencing is invalid for ${namespace}`);
  }
  const storedSource = decodeStoredBytes(values, sourceKeys);
  if (storedSource.byteLength !== sourceBytes.byteLength || storedSource.some((byte, index) => byte !== sourceBytes[index])) {
    throw error(`stored source bytes do not match the requested source for ${namespace}`);
  }
  if (receiptKeys.some((key, index) => key !== expectedReceipts[index]) || receiptKeys.length !== expectedReceipts.length) {
    throw error(`stored receipt chunk sequencing is invalid for ${namespace}`);
  }
  const receiptText = decodeStoredText(values, receiptKeys);
  if (expectedReceiptText !== undefined && receiptText !== expectedReceiptText) throw error(`stored receipt does not match the deterministic import for ${namespace}`);
  const receipt = parseStoredReceipt(receiptText);
  if (!receipt || receipt.sha256 !== sourceHash || receipt.sourceBytes !== sourceBytes.byteLength || receipt.target.appKey !== appKey || receipt.target.locale !== locale || receipt.target.pageId !== pageId) {
    throw error(`storage namespace collision for ${namespace}`);
  }
  if (receipt.sourceOriginal.length !== sourceKeys.length || receipt.sourceOriginal.some((key, index) => key !== sourceKeys[index])) {
    throw error(`stored source reference ledger is invalid for ${namespace}`);
  }
  return receipt;
}

function storedBlockStart(receipt: LegacyImportReceipt | undefined, pageIndex: number): number | undefined {
  if (!receipt) return undefined;
  const indexes: number[] = [];
  for (const disposition of receipt.fieldLevelDispositions) {
    for (const path of disposition.targetPaths ?? []) {
      const match = path.match(/^\/pages\/(\d+)\/blocks\/(\d+)\//);
      if (!match || Number(match[1]) !== pageIndex) continue;
      indexes.push(Number(match[2]));
    }
  }
  return indexes.length > 0 ? Math.min(...indexes) : undefined;
}

function parseSections(source: LegacyObject): LegacySection[] {
  const variant = source.variant;
  if (!isRecord(variant)) throw error('variant object is required');
  if (!Array.isArray(source.sections)) throw error('sections array is required');
  if (source.sections.length > MAX_SECTIONS) throw error(`sections exceed ${String(MAX_SECTIONS)}`);
  const keys = new Set<string>();
  const identities = new Set<string>();
  return source.sections.map((raw, index) => {
    if (!isRecord(raw)) throw error(`sections[${String(index)}] must be an object`);
    const sectionTypeValue = raw.section_type;
    if (typeof sectionTypeValue !== 'string' || !sectionTypeValue.trim()) throw error(`sections[${String(index)}].section_type is required`);
    const sectionType = sectionTypeValue.trim().toLowerCase();
    const keyValue = raw.key;
    if (keyValue !== undefined && (typeof keyValue !== 'string' || !keyValue.trim())) throw error(`sections[${String(index)}].key must be a non-empty string when present`);
    const key = typeof keyValue === 'string' ? keyValue : undefined;
    if (key && keys.has(key)) throw error(`duplicate section key ${key}`);
    if (key) keys.add(key);
    const orderValue = raw.order;
    if (orderValue !== undefined && (typeof orderValue !== 'number' || !Number.isSafeInteger(orderValue) || orderValue < 1)) throw error(`sections[${String(index)}].order must be a positive integer`);
    const order = typeof orderValue === 'number' ? orderValue : index + 1;
    const identity = key ? `key:${key}` : `type:${sectionType}:order:${String(order)}`;
    if (identities.has(identity)) throw error(`duplicate section identity ${identity}`);
    identities.add(identity);
    if (!isRecord(raw.content)) throw error(`sections[${String(index)}].content must be an object`);
    if (raw.enabled !== undefined && typeof raw.enabled !== 'boolean') throw error(`sections[${String(index)}].enabled must be boolean`);
    return { index, key, sectionType, content: raw.content, order, enabled: raw.enabled !== false };
  });
}

function sourceFieldPath(section: LegacySection, ...keys: Array<string | number>): string {
  return sourcePointer('sections', section.index, 'content', ...keys);
}

function storyBlock(
  section: LegacySection,
  blockIndex: number,
  pageIndex: number,
  mappings: Map<string, Mapping>,
): PresentationBlock | undefined {
  const content = section.content;
  const blockPointer = (field: string): string => targetPointer(pageIndex, blockIndex, 'content', 'product_story', ...field.split('/'));
  const pending = new Map<string, Mapping>();
  const headingSelection = selectTextField(content, ['title', 'heading'], sourcePointer('sections', section.index, 'content'), blockPointer('heading'), pending);
  const bodySelection = selectTextField(content, ['subtitle', 'description', 'body', 'caption', 'text'], sourcePointer('sections', section.index, 'content'), blockPointer('body'), pending);
  const heading = headingSelection?.value;
  const body = bodySelection?.value;
  const featureKey = Array.isArray(content.features) ? 'features' : Array.isArray(content.items) ? 'items' : undefined;
  const features = featureKey ? content[featureKey] : [];
  const items: PresentationProductStory['items'] = [];
  let firstAccepted: { sourceIndex: number; title: TextSelection; description: TextSelection } | undefined;
  if (!Array.isArray(features)) throw error(`sections[${String(section.index)}].${String(featureKey)} must be an array`);
  features.forEach((raw, index) => {
    if (!isRecord(raw)) return;
    const itemMappings = new Map<string, Mapping>();
    const itemIndex = items.length;
    const itemPointer = (field: string): string => targetPointer(pageIndex, blockIndex, 'content', 'product_story', 'items', itemIndex, field);
    const sourcePrefix = featureKey ? sourceFieldPath(section, featureKey, index) : sourceFieldPath(section, index);
    const titleSelection = selectTextField(raw, ['title', 'name'], sourcePrefix, itemPointer('title'), itemMappings);
    const descriptionSelection = selectTextField(raw, ['description', 'body', 'text'], sourcePrefix, itemPointer('description'), itemMappings);
    if (titleSelection && descriptionSelection) {
      items.push(create(PresentationStoryItemSchema, { title: titleSelection.value, description: descriptionSelection.value, visualRef: '', altText: '' }));
      mergeMappings(pending, itemMappings);
      if (!firstAccepted) {
        firstAccepted = { sourceIndex: index, title: titleSelection, description: descriptionSelection };
      }
    }
  });
  const resolvedHeading = heading ?? body ?? (items[0]?.title ?? '');
  const resolvedBody = body ?? heading ?? (items[0]?.description ?? '');
  if (!resolvedHeading || !resolvedBody) return undefined;
  if (!heading && bodySelection) addMapping(pending, sourceFieldPath(section, bodySelection.key), blockPointer('heading'), bodySelection.normalized);
  if (!body && headingSelection) addMapping(pending, sourceFieldPath(section, headingSelection.key), blockPointer('body'), headingSelection.normalized);
  if (!heading && !body && firstAccepted && featureKey) {
    addMapping(pending, sourceFieldPath(section, featureKey, firstAccepted.sourceIndex, firstAccepted.title.key), blockPointer('heading'), firstAccepted.title.normalized);
    addMapping(pending, sourceFieldPath(section, featureKey, firstAccepted.sourceIndex, firstAccepted.description.key), blockPointer('body'), firstAccepted.description.normalized);
  }
  if (items.length === 0) {
    items.push(create(PresentationStoryItemSchema, { title: resolvedHeading, description: resolvedBody, visualRef: '', altText: '' }));
    const headingSource = headingSelection ? sourceFieldPath(section, headingSelection.key) : bodySelection ? sourceFieldPath(section, bodySelection.key) : undefined;
    const bodySource = bodySelection ? sourceFieldPath(section, bodySelection.key) : headingSelection ? sourceFieldPath(section, headingSelection.key) : undefined;
    if (headingSource) addMapping(pending, headingSource, blockPointer('items/0/title'), headingSelection?.normalized ?? bodySelection?.normalized ?? false);
    if (bodySource) addMapping(pending, bodySource, blockPointer('items/0/description'), bodySelection?.normalized ?? headingSelection?.normalized ?? false);
  }
  mergeMappings(mappings, pending);
  const story = create(PresentationProductStorySchema, { heading: resolvedHeading, body: resolvedBody, items });
  return create(PresentationBlockSchema, {
    id: `legacy-${String(blockIndex)}-${String(section.index)}`,
    kind: 'product-story',
    version: 1,
    variant: 'three-column',
    content: create(PresentationBlockContentSchema, { value: { case: 'productStory', value: story } }),
  });
}

function faqBlock(
  section: LegacySection,
  blockIndex: number,
  pageIndex: number,
  mappings: Map<string, Mapping>,
): PresentationBlock | undefined {
  const blockPointer = (field: string): string => targetPointer(pageIndex, blockIndex, 'content', 'faq', ...field.split('/'));
  const pending = new Map<string, Mapping>();
  const itemKey = Array.isArray(section.content.items) ? 'items' : Array.isArray(section.content.faqs) ? 'faqs' : undefined;
  const rawItems = itemKey ? section.content[itemKey] : [];
  if (!Array.isArray(rawItems)) throw error(`sections[${String(section.index)}].${String(itemKey)} must be an array`);
  const items: PresentationFAQItem[] = [];
  let firstAccepted: { question: TextSelection; sourcePrefix: string } | undefined;
  rawItems.forEach((raw, index) => {
    if (!isRecord(raw)) return;
    const itemMappings = new Map<string, Mapping>();
    const itemIndex = items.length;
    const sourcePrefix = itemKey ? sourceFieldPath(section, itemKey, index) : sourceFieldPath(section, index);
    const question = selectTextField(raw, ['question', 'title'], sourcePrefix, blockPointer(`items/${String(itemIndex)}/question`), itemMappings);
    const answer = selectTextField(raw, ['answer', 'description', 'body'], sourcePrefix, blockPointer(`items/${String(itemIndex)}/answer`), itemMappings);
    if (!question || !answer) return;
    const accessibleSelection = textField(raw, 'accessible_label', `${sourcePrefix}/accessible_label`, blockPointer(`items/${String(itemIndex)}/accessible_label`), itemMappings);
    const accessibleLabel = accessibleSelection ?? question.value;
    if (accessibleSelection === undefined) addMapping(itemMappings, `${sourcePrefix}/${escapePointerToken(question.key)}`, blockPointer(`items/${String(itemIndex)}/accessible_label`), question.normalized);
    items.push(create(PresentationFAQItemSchema, { question: question.value, answer: answer.value, accessibleLabel }));
    mergeMappings(pending, itemMappings);
    if (!firstAccepted) firstAccepted = { question, sourcePrefix };
  });
  const headingSelection = selectTextField(section.content, ['title', 'heading'], sourcePointer('sections', section.index, 'content'), blockPointer('heading'), pending);
  const heading = headingSelection?.value ?? items[0]?.question ?? '';
  if (!heading || items.length === 0) return undefined;
  if (!headingSelection && firstAccepted) {
    addMapping(pending, `${firstAccepted.sourcePrefix}/${escapePointerToken(firstAccepted.question.key)}`, blockPointer('heading'), firstAccepted.question.normalized);
  }
  mergeMappings(mappings, pending);
  return create(PresentationBlockSchema, {
    id: `legacy-${String(blockIndex)}-${String(section.index)}`,
    kind: 'faq',
    version: 1,
    variant: 'accordion',
    content: create(PresentationBlockContentSchema, { value: { case: 'faq', value: create(PresentationFAQSchema, { heading, items }) } }),
  });
}

function closingBlock(
  section: LegacySection,
  blockIndex: number,
  pageIndex: number,
  unavailableReason: string,
  mappings: Map<string, Mapping>,
): PresentationBlock | undefined {
  const pending = new Map<string, Mapping>();
  const blockPointer = (field: string): string => targetPointer(pageIndex, blockIndex, 'content', 'closing_action', ...field.split('/'));
  const headingSelection = selectTextField(section.content, ['cta_text', 'title', 'heading', 'label'], sourcePointer('sections', section.index, 'content'), blockPointer('heading'), pending);
  if (!headingSelection) return undefined;
  const descriptionSelection = selectTextField(section.content, ['description', 'subtitle', 'body'], sourcePointer('sections', section.index, 'content'), blockPointer('description'), pending);
  const heading = headingSelection.value;
  const description = descriptionSelection?.value ?? heading;
  if (!descriptionSelection) addMapping(pending, sourceFieldPath(section, headingSelection.key), blockPointer('description'), headingSelection.normalized);
  const action = create(PresentationActionSchema, {
    kind: 'unavailable',
    label: heading,
    accessibleLabel: heading,
    reason: unavailableReason,
  });
  addMapping(pending, sourceFieldPath(section, headingSelection.key), blockPointer('actions/0/label'), headingSelection.normalized);
  addMapping(pending, sourceFieldPath(section, headingSelection.key), blockPointer('actions/0/accessible_label'), headingSelection.normalized);
  mergeMappings(mappings, pending);
  return create(PresentationBlockSchema, {
    id: `legacy-${String(blockIndex)}-${String(section.index)}`,
    kind: 'closing-action',
    version: 1,
    variant: 'plain',
    content: create(PresentationBlockContentSchema, { value: { case: 'closingAction', value: create(PresentationClosingActionSchema, { heading, description, actions: [action], visualRef: '' }) } }),
  });
}

function pageAndTarget(document: ProductPresentationDocument, options: LegacyImportOptions) {
  if (options.adapterVersion !== ADAPTER_VERSION) throw error('unsupported adapter version');
  if (!options.appKey || !options.locale) throw error('appKey and locale are required');
  const matchingApps = document.apps.filter((app) => app.key === options.appKey);
  if (matchingApps.length !== 1) throw error('target app must identify exactly one existing app');
  const app = matchingApps[0];
  if (!app || app.enabled || app.visibility !== 'private' || app.publication !== 'draft') throw error('target app must be disabled, private, and draft');
  const bundlePageIds = [document.bundle?.pageId, document.bundle?.emptyPageId].filter((pageId): pageId is string => Boolean(pageId));
  if (!app.pageId || bundlePageIds.includes(app.pageId)) throw error('target page must not alias a bundle page');
  const owners = document.apps.filter((candidate) => candidate.pageId === app.pageId);
  if (owners.length !== 1 || owners[0]?.key !== app.key) throw error('target page must have exactly one owner');
  const pages = document.pages.filter((page) => page.id === app.pageId && page.locale === options.locale);
  if (pages.length !== 1) throw error('target must have exactly one page for the exact locale');
  const page = pages[0];
  const unavailableReason = page?.display?.shell?.unavailableReason.trim() ?? '';
  if (!page || !unavailableReason) throw error('target page must have a configured unavailable reason');
  return { app, page, unavailableReason };
}

function receiptFor(
  sourceBytes: number,
  sourceHash: string,
  sourceOriginal: string[],
  appKey: string,
  locale: string,
  pageId: string,
  source: LegacyObject,
  mappings: Map<string, Mapping>,
): LegacyImportReceipt {
  const leaves: string[] = [];
  walkLeaves(source, '', leaves);
  const fieldLevelDispositions: LegacyFieldDisposition[] = leaves.sort().map((sourcePath): LegacyFieldDisposition => {
    const mapping = mappings.get(sourcePath);
    if (!mapping) return { sourcePath, disposition: 'preserved-only', reason: 'not activated by adapter v1' };
    return {
      sourcePath,
      disposition: mapping.normalized ? 'normalized' : 'mapped',
      targetPaths: [...mapping.targetPaths],
      ...(mapping.normalized ? { normalization: 'trim and collapse control/whitespace runs' } : {}),
    };
  });
  return {
    receiptVersion: 1,
    adapterVersion: 1,
    sha256: `sha256:${sourceHash}`,
    sourceBytes,
    sourceOriginal,
    target: { appKey, locale, pageId },
    fieldLevelDispositions,
  };
}

/** Convert one explicit legacy snapshot into a private draft document without network or storage side effects. */
export async function importLegacyPresentationSnapshot(
  document: ProductPresentationDocument,
  sourceText: string,
  options: LegacyImportOptions,
): Promise<LegacyImportResult> {
  if (typeof sourceText !== 'string' || !sourceText) throw error('source text is required');
  const sourceBytes = new TextEncoder().encode(sourceText);
  if (sourceBytes.byteLength > MAX_SOURCE_BYTES) throw error('source exceeds the 1 MiB limit');
  try {
    const roundTrip = new TextDecoder('utf-8', { fatal: true }).decode(sourceBytes);
    if (roundTrip !== sourceText) throw error('source is not a well-formed UTF-8 round trip');
  } catch (cause) {
    if (cause instanceof Error && cause.message.startsWith('Legacy presentation import:')) throw cause;
    throw error('source is not a well-formed UTF-8 round trip');
  }
  rejectDuplicateKeys(sourceText);
  let parsed: unknown;
  try {
    parsed = JSON.parse(sourceText);
  } catch {
    throw error('source is malformed JSON');
  }
  if (!isRecord(parsed)) throw error('source envelope must be an object');
  const sections = parseSections(parsed);
  const target = pageAndTarget(document, options);
  const sourceHash = await sha256(sourceBytes);
  const working = cloneDocument(document);
  const workingPage = working.pages.find((page) => page.id === target.page.id && page.locale === options.locale);
  if (!workingPage) throw error('cloned target page is missing');
  const workingTarget = { ...target, page: workingPage };
  const strings = working.strings[options.locale] ?? create(PresentationStringTableSchema, { values: {} });
  const namespace = await namespaceFor(options.appKey, options.locale, sourceHash);
  const namespacePrefix = `${namespace}-`;
  const hasStoredNamespace = Object.keys(strings.values).some((key) => key.startsWith(namespacePrefix));
  const stored = hasStoredNamespace
    ? existingImport(strings.values, namespace, sourceBytes, `sha256:${sourceHash}`, options.appKey, options.locale, workingTarget.page.id)
    : undefined;

  const mappings = new Map<string, Mapping>();
  const importedBlocks: PresentationBlock[] = [];
  const sortedSections = [...sections].sort((left, right) => left.order - right.order || left.index - right.index);
  const pageIndex = working.pages.indexOf(workingTarget.page);
  if (pageIndex < 0) throw error('cloned target page index is invalid');
  const blockStart = storedBlockStart(stored, pageIndex) ?? workingTarget.page.blocks.length;
  for (const section of sortedSections) {
    if (!section.enabled) continue;
    const blockIndex = blockStart + importedBlocks.length;
    const block = section.sectionType === 'faq'
      ? faqBlock(section, blockIndex, pageIndex, mappings)
      : section.sectionType === 'cta' || section.sectionType === 'download'
        ? closingBlock(section, blockIndex, pageIndex, target.unavailableReason, mappings)
        : section.sectionType === 'hero' || section.sectionType === 'features' || section.sectionType === 'video'
          ? storyBlock(section, blockIndex, pageIndex, mappings)
          : undefined;
    if (block) importedBlocks.push(block);
  }
  const receipt = receiptFor(sourceBytes.byteLength, sourceHash, [], options.appKey, options.locale, workingTarget.page.id, parsed, mappings);
  const sourceChunkSet = chunks(sourceBytes, `${namespace}-source`);
  receipt.sourceOriginal.push(...sourceChunkSet.keys);
  const receiptBytes = new TextEncoder().encode(JSON.stringify(receipt));
  const receiptChunkSet = chunks(receiptBytes, `${namespace}-receipt`);
  const prior = stored
    ? existingImport(strings.values, namespace, sourceBytes, `sha256:${sourceHash}`, options.appKey, options.locale, workingTarget.page.id, JSON.stringify(receipt))
    : undefined;
  if (prior) return { document: working, receipt: prior };
  const allValues = { ...sourceChunkSet.values, ...receiptChunkSet.values };
  for (const key of Object.keys(allValues)) {
    if (key in strings.values) throw error(`storage value collision for ${key}`);
  }
  const existingIDs = new Set(workingTarget.page.blocks.map((block) => block.id));
  for (const block of importedBlocks) {
    if (existingIDs.has(block.id)) throw error(`block ID collision for ${block.id}`);
    existingIDs.add(block.id);
  }
  workingTarget.page.blocks.push(...importedBlocks);
  strings.values = { ...strings.values, ...allValues };
  working.strings[options.locale] = strings;
  if (new TextEncoder().encode(formatPresentationDocument(working)).byteLength > MAX_DOCUMENT_BYTES) {
    throw error('result exceeds the 8 MiB document limit');
  }
  return { document: working, receipt };
}
