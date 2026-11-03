import { type MessageShape } from '@bufbuild/protobuf';
import { GraphSchema } from '@vrooli/proto-types/system-monitor/v1/devicegraph/devicegraph_pb';
import { parseProtoMessage, parseMetricsResponse } from './proto-contracts';

export { parseMetricsResponse };

export function parseDeviceGraph(data: unknown): MessageShape<typeof GraphSchema> {
  return parseProtoMessage(GraphSchema, data);
}
