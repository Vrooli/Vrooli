import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { DesktopRecordsResponseSchema } from "@vrooli/proto-types/scenario-to-desktop/v1/domain/records_pb";
import { presentDesktopRecords } from "./recordPresentation";

describe("presentDesktopRecords", () => {
  it("preserves record placement and screen-recording evidence for presentation", () => {
    const response = create(DesktopRecordsResponseSchema, {
      records: [
        {
          record: {
            id: "record-1",
            buildId: "build-1",
            scenarioName: "demo",
            outputPath: "/tmp/demo",
            createdAt: { seconds: 1_700_000_000n, nanos: 0 },
          },
          hasBuild: true,
          buildState: "ready",
          smokeTestId: "smoke-1",
          screenRecording: {
            recorded: true,
            durationMs: 12_345n,
            fileSizeBytes: 6_789n,
          },
        },
      ],
    });

    const [record] = presentDesktopRecords(response);
    expect(record?.record).toEqual({
      id: "record-1",
      build_id: "build-1",
      scenario_name: "demo",
      output_path: "/tmp/demo",
      created_at: "2023-11-14T22:13:20.000Z",
    });
    expect(record?.smoke_test_id).toBe("smoke-1");
    expect(record?.screen_recording).toEqual({
      recorded: true,
      duration_ms: 12_345,
      file_size_bytes: 6_789,
      error: undefined,
    });
  });
});
