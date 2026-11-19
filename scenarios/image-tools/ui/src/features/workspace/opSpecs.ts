import { OP_SPECS } from "./opSpecs.generated";
import type { OpSpec } from "./opSpecTypes";

/**
 * Deterministic operation forms.
 *
 * The data lives in `opSpecs.generated.ts`, rendered by `make capability-gen`
 * from the capability manifests (presentation.fields) and cross-checked against
 * the proto `OpParams` descriptor. Adding a parameter is a manifest edit; the
 * generator fails if it names a field the wire does not carry.
 *
 * Only the operation list remains hand-shaped: a deterministic op with no
 * declared form is simply absent, which keeps advanced/CLI-first transforms out
 * of the Inspector without a second exclusion list.
 */
export { OP_SPECS };
export type { ControlKind, OpField, OpSpec } from "./opSpecTypes";
export { ENCODE_FORMATS } from "./opSpecTypes";

/** Spec for an operation name, or `undefined` if the op has no form. */
export const opSpec = (operation: string): OpSpec | undefined => OP_SPECS[operation];
