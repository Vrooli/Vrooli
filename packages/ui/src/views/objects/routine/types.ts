import { RoutineVersion } from "@shared/consts";
import { CreateProps, UpdateProps, ViewProps } from "../types";

export interface RoutineCreateProps extends CreateProps<RoutineVersion> {
    isSubroutine?: boolean;
}
export interface RoutineUpdateProps extends UpdateProps<RoutineVersion> {}
export interface RoutineViewProps extends ViewProps<RoutineVersion> {}