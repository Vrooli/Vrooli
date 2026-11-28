import { z } from 'zod';
import { parseLandingConfigJson } from '../../shared/api/landing';
import { isValidVisitorId } from '../../shared/lib/visitorIdentity';

const envelope = z.object({
  request: z.object({ route: z.string(), locale: z.string(), variant: z.string() }).strict(),
  canonicalBaseUrl: z.string().optional(), config: z.record(z.unknown()),
  visitorId: z.string().refine(isValidVisitorId).optional(),
}).strict();

export function readPresentationBootstrap(request: { route: string; locale: string; variant: string }) {
  const script = document.getElementById('lpbs-presentation-bootstrap');
  if (script?.getAttribute('type') !== 'application/json' || !script.textContent) return undefined;
  try {
    const value = envelope.parse(JSON.parse(script.textContent));
    if (value.request.route !== request.route || value.request.locale !== request.locale || value.request.variant !== request.variant) return undefined;
    const config = parseLandingConfigJson(JSON.stringify(value.config));
    const d = config.presentation.diagnostics;
    if (!d || d.preview || d.requestedRoute !== request.route || d.resolvedRoute !== request.route || d.requestedVariant !== request.variant) return undefined;
    return { config, canonicalBaseUrl: value.canonicalBaseUrl ?? '', visitorId: value.visitorId };
  } catch { return undefined; }
}
