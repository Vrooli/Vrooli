/**
 * Test factory for the registry Node proto message. Lives with the fleet
 * feature (not in the cross-domain test-utils) so deleting the feature folder
 * takes its doubles along. `makeNode()` returns a TRUSTED online node by
 * default; tests override the fields the case is about.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  NodeSchema,
  NodeStatus,
  type Node,
} from "@vrooli/proto-types/vrooli-bridge/v1/registry/registry_pb";

export const makeNode = (overrides: MessageInitShape<typeof NodeSchema> = {}): Node =>
  create(NodeSchema, {
    id: "node-1",
    name: "mac-mini-office",
    os: "darwin",
    arch: "arm64",
    revision: "abc1234567def",
    status: NodeStatus.ONLINE,
    online: true,
    ...overrides,
  });
