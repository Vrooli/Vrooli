import { ApiVersion } from "@shared/consts";
import { CreateProps, UpdateProps, ViewProps } from "../types";

export interface ApiCreateProps extends CreateProps<ApiVersion> {}
export interface ApiUpdateProps extends UpdateProps<ApiVersion> {}
export interface ApiViewProps extends ViewProps<ApiVersion> {}