import { clone, fromJson, type JsonValue } from '@bufbuild/protobuf';
import { LandingConfigResponseSchema } from '@vrooli/proto-types/landing-page-react-vite/v1/config_pb';
import type { LandingConfigResponse } from '../api';
import rawFallback from '../../../../.vrooli/variants/fallback.json';
import { normalizeHeaderConfig } from './headerConfig';

interface FallbackJson {
  variant: { id?: number; slug?: string; name?: string; description?: string; axes?: Record<string, string> };
  axes?: Record<string, string>;
  sections?: unknown[];
  pricing?: unknown;
  downloads?: unknown[];
  header?: unknown;
}

// The fallback payload is authored as REST-style JSON (snake_case fields). It is
// parsed into the LandingConfigResponse proto with fromJson (which accepts the
// original snake_case field names), then the header is normalized. Rendered when
// the live API is unavailable.
const FALLBACK_CONFIG: LandingConfigResponse = (() => {
  const payload = rawFallback as FallbackJson;
  const variantJson = {
    ...payload.variant,
    axes: payload.variant?.axes ?? payload.axes ?? {},
  };

  // Pricing is intentionally omitted: the authored fallback pricing is REST-shaped
  // (plain-JSON plan metadata + string billing intervals) and cannot round-trip
  // through the proto JSON decoder's common.v1.JsonValue maps / enums. With pricing
  // absent, the pricing surface renders its demo placeholders — the offline-safe
  // behavior this fallback exists to provide.
  const configJson: Record<string, unknown> = {
    variant: variantJson,
    sections: payload.sections ?? [],
    downloads: payload.downloads ?? [],
    header: payload.header ?? {},
    fallback: true,
  };

  const parsed = fromJson(LandingConfigResponseSchema, configJson as JsonValue, {
    ignoreUnknownFields: true,
  });
  parsed.header = normalizeHeaderConfig(parsed.header ?? undefined, parsed.variant?.name);
  return parsed;
})();

export function getFallbackLandingConfig(): LandingConfigResponse {
  return clone(LandingConfigResponseSchema, FALLBACK_CONFIG);
}
