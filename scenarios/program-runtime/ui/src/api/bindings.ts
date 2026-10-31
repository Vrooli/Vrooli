import { fromJson, type JsonValue } from "@bufbuild/protobuf";
import { buildApiUrl } from "@vrooli/api-base";
import {
  ListBindingsResponseSchema,
  ListUnboundResponseSchema,
  type Binding,
  type UnboundCapability,
} from "@vrooli/proto-types/program-runtime/v1/bindings/bindings_pb";

import { API_BASE, PROTO_READ_OPTIONS, decodeApiError } from "./client";

const bindingPath = "/vrooli.program_runtime.v1.bindings.BindingRegistryService";

async function post<T>(procedure: string, body: object, schema: Parameters<typeof fromJson>[0]): Promise<T> {
  const res = await fetch(buildApiUrl(`${bindingPath}/${procedure}`, { baseUrl: API_BASE }), {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(body),
    cache: "no-store",
  });
  if (!res.ok) throw await decodeApiError(res);
  return fromJson(schema as never, (await res.json()) as JsonValue, PROTO_READ_OPTIONS) as T;
}

export async function fetchBindings(): Promise<Binding[]> {
  const response = await post<{ bindings: Binding[] }>("ListBindings", {}, ListBindingsResponseSchema);
  return response.bindings;
}

export async function fetchUnbound(): Promise<UnboundCapability[]> {
  const response = await post<{ capabilities: UnboundCapability[] }>("ListUnbound", {}, ListUnboundResponseSchema);
  return response.capabilities;
}
