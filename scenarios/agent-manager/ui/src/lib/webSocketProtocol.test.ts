import { describe, expect, it } from "vitest";
import { parseWebSocketMessage } from "./webSocketProtocol";

describe("workflow lifecycle websocket projection", () => {
  it("accepts metadata without inventing payload fields", () => {
    const message = parseWebSocketMessage({
      type: "AGENT_MANAGER_WS_MESSAGE_TYPE_WORKFLOW_LIFECYCLE",
      workflow_lifecycle: {
        execution_id: "1c7b2c39-bd80-4cd7-a5b2-b55f84ee3bd7",
        definition_digest: "sha256:abc",
        status: "waiting",
        node_id: "approval",
        journal_sequence: "4",
        journal_payload_digest: "sha256:def",
      },
    });
    expect(message?.type).toBe("workflow_lifecycle");
    expect(message?.payload).toMatchObject({ executionId: "1c7b2c39-bd80-4cd7-a5b2-b55f84ee3bd7", nodeId: "approval" });
    expect(message?.payload).not.toHaveProperty("prompt");
    expect(message?.payload).not.toHaveProperty("result");
  });
});
