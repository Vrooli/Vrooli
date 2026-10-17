import { createClient } from '@connectrpc/connect';
import { AIService } from '@vrooli/proto-types/browser-automation-studio/v1/ai/ai_pb';
import type {
  ElementInfo as ProtoElementInfo,
  ElementSelectionResult as ProtoElementSelectionResult,
  ElementHierarchyEntry as ProtoElementHierarchyEntry,
  SelectorOption as ProtoSelectorOption,
  Rectangle as ProtoRectangle,
} from '@vrooli/proto-types/browser-automation-studio/v1/ai/ai_pb';
import type {
  ElementInfo,
  ElementHierarchyEntry,
  ElementCoordinateResponse,
  SelectorOption,
  BoundingBox,
} from '@/types/elements';
import { transport } from './client';

export const aiClient = createClient(AIService, transport);

// =============================================================================
// Proto → legacy-shape mappers
// =============================================================================
// The UI was originally typed against the REST JSON shape that mirrored the
// Go structs in api/handlers/ai/types.go. To avoid a big-bang rewrite of every
// caller during the Phase 9 migration, we map the proto response back onto
// those legacy shapes at the API boundary. New callers should prefer the
// proto types directly.

const mapRectangle = (r?: ProtoRectangle): BoundingBox => ({
  x: r?.x ?? 0,
  y: r?.y ?? 0,
  width: r?.width ?? 0,
  height: r?.height ?? 0,
});

const mapSelector = (s: ProtoSelectorOption): SelectorOption => ({
  selector: s.selector,
  type: s.type,
  robustness: s.robustness,
  fallback: s.fallback,
});

export const mapProtoElementInfo = (e?: ProtoElementInfo | null): ElementInfo | null => {
  if (!e) return null;
  return {
    text: e.text,
    tagName: e.tagName,
    type: e.type,
    selectors: e.selectors.map(mapSelector),
    boundingBox: mapRectangle(e.boundingBox),
    confidence: e.confidence,
    category: e.category,
    attributes: { ...e.attributes },
  };
};

export const mapProtoHierarchyEntry = (entry: ProtoElementHierarchyEntry): ElementHierarchyEntry => ({
  element: mapProtoElementInfo(entry.element) ?? {
    text: '',
    tagName: '',
    type: '',
    selectors: [],
    boundingBox: { x: 0, y: 0, width: 0, height: 0 },
    confidence: 0,
    category: '',
    attributes: {},
  },
  selector: entry.selector,
  depth: entry.depth,
  path: [...entry.path],
  pathSummary: entry.pathSummary,
});

export const mapProtoSelectionResult = (
  s?: ProtoElementSelectionResult,
): ElementCoordinateResponse => ({
  element: mapProtoElementInfo(s?.element ?? undefined),
  candidates: (s?.candidates ?? []).map(mapProtoHierarchyEntry),
  selectedIndex: s?.selectedIndex ?? 0,
});
