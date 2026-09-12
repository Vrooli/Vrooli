import { createClient } from "@connectrpc/connect";
import { SketchService } from "@vrooli/proto-types/react-component-library/v1/sketch/sketch_pb";
import { transport } from "./client";

// All workspace reads and writes use the same generated, routed service as CLI.
export const sketchClient = createClient(SketchService, transport);
