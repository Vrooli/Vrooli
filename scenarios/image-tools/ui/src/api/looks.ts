import { createClient } from "@connectrpc/connect";
import {
  LookKind,
  LooksService,
  StepKind,
  type CompileLookResponse,
  type ListLooksResponse,
  type Look,
  type LookStep,
  type RenderPreviewResponse,
} from "@vrooli/proto-types/image-tools/v1/looks/looks_pb";

import { transport } from "./client";

/** Connect-Web client for LooksService (the Look/Style library). */
export const looksClient = createClient(LooksService, transport);

export { LookKind, StepKind };
export type { Look, LookStep, ListLooksResponse, CompileLookResponse, RenderPreviewResponse };
