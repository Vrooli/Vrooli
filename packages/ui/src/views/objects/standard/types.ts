import { StandardVersion } from "@shared/consts";
import { UpsertProps, ViewProps } from "../types";

export interface StandardUpsertProps extends UpsertProps<StandardVersion> { }
export interface StandardViewProps extends ViewProps<StandardVersion> { }