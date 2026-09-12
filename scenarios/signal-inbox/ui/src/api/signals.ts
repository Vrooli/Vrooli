import { createClient } from "@connectrpc/connect";
import {
  SignalsService,
  UploadImageResponseSchema,
  type CaptureSignalRequest,
} from "../../../../../packages/proto/gen/typescript/signal-inbox/v1/signals/signals_pb";
import type { Signal } from "../../../../../packages/proto/gen/typescript/signal-inbox/v1/shared/signals_pb";

import { transport } from "./client";
import { decodeApiError, fromJson, PROTO_READ_OPTIONS, uploadFile } from "./client";

export const signalsClient = createClient(SignalsService, transport);

export async function uploadSignalImage(file: File): Promise<string> {
  const formData = new FormData();
  formData.set("file", file, file.name);
  const response = await uploadFile("/signals/images", formData);
  if (!response.ok) {
    throw await decodeApiError(response);
  }
  const decoded = fromJson(UploadImageResponseSchema, await response.json(), PROTO_READ_OPTIONS);
  if (!decoded.image?.payloadRef) {
    throw new Error("image upload returned no payload reference");
  }
  return decoded.image.payloadRef;
}
export type { CaptureSignalRequest, Signal };
