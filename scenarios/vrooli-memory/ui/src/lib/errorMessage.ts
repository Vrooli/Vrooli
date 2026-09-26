import { Code, ConnectError } from "@connectrpc/connect";
import type { TFunction } from "i18next";

import { ApiError } from "../api/client";
import { strings } from "../consts/strings";

const CONNECT_ERROR_KEYS = {
  [Code.Canceled]: strings.errors.canceled,
  [Code.Unknown]: strings.errors.unknown,
  [Code.InvalidArgument]: strings.errors.invalid_argument,
  [Code.DeadlineExceeded]: strings.errors.deadline_exceeded,
  [Code.NotFound]: strings.errors.not_found,
  [Code.AlreadyExists]: strings.errors.already_exists,
  [Code.PermissionDenied]: strings.errors.permission_denied,
  [Code.ResourceExhausted]: strings.errors.resource_exhausted,
  [Code.FailedPrecondition]: strings.errors.failed_precondition,
  [Code.Aborted]: strings.errors.aborted,
  [Code.OutOfRange]: strings.errors.out_of_range,
  [Code.Unimplemented]: strings.errors.unimplemented,
  [Code.Internal]: strings.errors.internal,
  [Code.Unavailable]: strings.errors.unavailable,
  [Code.DataLoss]: strings.errors.data_loss,
  [Code.Unauthenticated]: strings.errors.unauthenticated,
} as const satisfies Record<Code, string>;

type ErrorKey = (typeof CONNECT_ERROR_KEYS)[Code];

const API_ERROR_CODE_KEYS: Record<string, ErrorKey> = {
  canceled: strings.errors.canceled,
  unknown: strings.errors.unknown,
  invalid_argument: strings.errors.invalid_argument,
  invalid_request: strings.errors.invalid_argument,
  deadline_exceeded: strings.errors.deadline_exceeded,
  not_found: strings.errors.not_found,
  already_exists: strings.errors.already_exists,
  permission_denied: strings.errors.permission_denied,
  resource_exhausted: strings.errors.resource_exhausted,
  failed_precondition: strings.errors.failed_precondition,
  aborted: strings.errors.aborted,
  out_of_range: strings.errors.out_of_range,
  unimplemented: strings.errors.unimplemented,
  internal: strings.errors.internal,
  unavailable: strings.errors.unavailable,
  data_loss: strings.errors.data_loss,
  unauthenticated: strings.errors.unauthenticated,
};

const normalizeApiErrorCode = (code: string): ErrorKey => {
  return API_ERROR_CODE_KEYS[code] ?? strings.errors.unknown;
};

export function errorMessage(err: unknown, t: TFunction): string {
  if (err instanceof ConnectError) {
    return t(CONNECT_ERROR_KEYS[err.code], { message: err.rawMessage });
  }
  if (err instanceof ApiError) {
    return t(normalizeApiErrorCode(err.code), { message: err.message });
  }
  if (err instanceof Error) {
    return err.message;
  }
  return String(err);
}
