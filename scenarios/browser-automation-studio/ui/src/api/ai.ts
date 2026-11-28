import { Code, ConnectError, createClient } from '@connectrpc/connect';
import { create } from '@bufbuild/protobuf';
import { AIAnalyzeElementsRequestSchema, AIService } from '@vrooli/proto-types/browser-automation-studio/v1/ai/ai_pb';
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

export const newGatewayRootOperationID = (): string => {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (value) => value.toString(16).padStart(2, '0')).join('');
};

export const analyzeElementsWithAI = (request: { url: string; intent: string }, rootOperationID: string) => {
  const headers = new Headers({ 'Idempotency-Key': rootOperationID });
  return aiClient.aIAnalyzeElements(create(AIAnalyzeElementsRequestSchema, request), { headers });
};

export const aiRequestErrorMessage = (error: unknown, fallback: string): string => {
  if (
    error instanceof ConnectError
    && error.code === Code.FailedPrecondition
    && error.metadata.get('X-Vrooli-AI-Gateway-Error-Code') === 'INFERENCE_ERROR_CODE_NO_ELIGIBLE_ROUTE'
  ) {
    const source = error.metadata.get('X-Vrooli-AI-Source');
    const fundingBoundary = 'No inference was dispatched, and funding did not change.';
    switch (source) {
      case 'local':
        return `No local route is eligible for this request. In Settings → Confidential data sharing, choose a configured remote source and grant consent, or ask the account owner to enable or upgrade an eligible route. ${fundingBoundary}`;
      case 'own-key':
        return `Your selected OpenRouter source has no eligible route for this request. Check the key and route in Settings → AI Settings, or choose another configured source in Settings → Confidential data sharing. If the account route is unavailable, ask the account owner to enable or upgrade it. ${fundingBoundary}`;
      case 'vrooli-credits':
        return `Vrooli AI credits has no eligible route for this request. Ask the account owner to enable or upgrade an eligible route, or choose your configured OpenRouter source in Settings → Confidential data sharing. ${fundingBoundary}`;
      default:
        return `No eligible route for the selected AI source. Choose another configured source in Settings → Confidential data sharing, or ask the account owner to enable or upgrade an eligible route. ${fundingBoundary}`;
    }
  }
  return error instanceof Error ? error.message : fallback;
};

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
