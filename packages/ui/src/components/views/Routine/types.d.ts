import { Routine } from "types";
import { CreateProps, UpdateProps, ViewProps } from "../types";

export interface RoutineCreateProps extends CreateProps<Routine> {
    isSubroutine?: boolean;
}
export interface RoutineUpdateProps extends UpdateProps<Routine> {}
export interface RoutineViewProps extends ViewProps<Routine> {}