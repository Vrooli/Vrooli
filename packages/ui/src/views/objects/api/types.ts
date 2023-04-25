import { ApiVersion } from "@local/shared";
import { UpsertProps, ViewProps } from "../types";

export interface ApiUpsertProps extends UpsertProps<ApiVersion> { }
export interface ApiViewProps extends ViewProps<ApiVersion> { }