import { ApiVersion } from "@shared/consts";
import { UpsertProps, ViewProps } from "../types";

export interface ApiUpsertProps extends UpsertProps<ApiVersion> { }
export interface ApiViewProps extends ViewProps<ApiVersion> { }