import { StandardVersion } from "@shared/consts";
import { CreateProps, UpdateProps, ViewProps } from "../types";

export interface StandardCreateProps extends CreateProps<StandardVersion> {}
export interface StandardUpdateProps extends UpdateProps<StandardVersion> {}
export interface StandardViewProps extends ViewProps<StandardVersion> {}