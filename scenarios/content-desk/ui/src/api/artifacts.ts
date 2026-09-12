import { createClient } from "@connectrpc/connect";
import { ArtifactsService } from "@vrooli/proto-types/content-desk/v1/artifacts/artifacts_pb";
import { transport } from "./client";

export const artifactsClient = createClient(ArtifactsService, transport);
