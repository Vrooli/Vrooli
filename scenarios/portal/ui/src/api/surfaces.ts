import { createClient } from "@connectrpc/connect";
import { SurfaceCatalogService } from "@vrooli/proto-types/portal/v1/surfaces/surfaces_pb";
import { transport } from "./client";

const catalog = createClient(SurfaceCatalogService, transport);
export const listSurfaces = (signal?: AbortSignal) => catalog.list({}, { signal });
export { SurfaceKind, SurfaceCapabilityState } from "@vrooli/proto-types/common/v1/surface_pb";
export type { SurfaceRef, SurfaceDescriptor } from "@vrooli/proto-types/common/v1/surface_pb";
