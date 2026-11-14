import type { OpParamValues } from "../../api/ops";
import { opSpec } from "./opSpecs";

/** Build the controlled-form default values for an op from its spec. */
export const defaultParamsFor = (operation: string): OpParamValues => {
  const spec = opSpec(operation);
  const values: OpParamValues = {};
  for (const field of spec?.fields ?? []) {
    values[field.name] = field.default;
  }
  return values;
};

/**
 * Coerce controlled values to the protojson shape the server expects.
 * `target_bytes` is an int64 → protojson encodes it as a string; everything
 * else maps 1:1.
 */
export const toRequestParams = (values: OpParamValues): OpParamValues => {
  const out: OpParamValues = {};
  for (const [key, value] of Object.entries(values)) {
    out[key] = key === "target_bytes" ? String(value) : value;
  }
  return out;
};
