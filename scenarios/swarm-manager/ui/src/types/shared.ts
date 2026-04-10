/**
 * Shared type helpers used across domain type files.
 */

import type { Message } from "@bufbuild/protobuf";

export type ProtoMessage<T extends Message> = Omit<T, "$typeName" | "$unknown">;
