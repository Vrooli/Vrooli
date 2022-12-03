import { Standard } from "types";
import { CreateProps, UpdateProps, ViewProps } from "../types";

export interface StandardCreateProps extends CreateProps<Standard> {}
export interface StandardUpdateProps extends UpdateProps<Standard> {}
export interface StandardViewProps extends ViewProps<Standard> {}