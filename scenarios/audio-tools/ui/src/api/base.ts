import { resolveApiBase } from "@vrooli/api-base";

export const API_BASE = resolveApiBase();
export const REST_API_BASE = resolveApiBase({ appendSuffix: true });
