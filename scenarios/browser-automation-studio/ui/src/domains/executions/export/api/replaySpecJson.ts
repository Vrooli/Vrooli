import { toJson } from "@bufbuild/protobuf";
import type { ReplaySpec } from "@vrooli/generated-proto/browser-automation-studio/v1/exports/exports_pb";
import { ReplaySpecSchema } from "@vrooli/generated-proto/browser-automation-studio/v1/exports/exports_pb";

type ExportPayloadWithReplaySpec = {
  movie_spec?: ReplaySpec;
};

export const serializeExportPayload = <T extends ExportPayloadWithReplaySpec>(payload: T): string => {
  const { movie_spec: movieSpec, ...metadata } = payload;
  return JSON.stringify({
    ...metadata,
    ...(movieSpec
      ? {
          movie_spec: toJson(ReplaySpecSchema, movieSpec, { useProtoFieldName: true }),
        }
      : {}),
  });
};
