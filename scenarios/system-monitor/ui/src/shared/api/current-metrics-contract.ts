import { fromJson, type JsonValue, type MessageShape } from '@bufbuild/protobuf';
import { MetricsResponseSchema } from '@vrooli/proto-types/system-monitor/v1/metrics/metrics_pb';

const PROTO_JSON_OPTIONS = { ignoreUnknownFields: true } as const;

export function parseMetricsResponse(data: unknown): MessageShape<typeof MetricsResponseSchema> {
  return fromJson(MetricsResponseSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}
