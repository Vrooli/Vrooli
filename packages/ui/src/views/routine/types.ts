import { RoutineVersion } from "@shared/consts";
import { UpsertProps, ViewProps } from "../types";

export interface RoutineUpsertProps extends UpsertProps<RoutineVersion> {
    isSubroutine?: boolean;
}
export interface RoutineViewProps extends ViewProps<RoutineVersion> { }