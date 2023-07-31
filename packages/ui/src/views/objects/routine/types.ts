import { RoutineVersion } from "@local/shared";
import { UpsertProps, ViewProps } from "../types";

export interface RoutineUpsertProps extends UpsertProps<RoutineVersion> {
    isSubroutine?: boolean;
}
export type RoutineViewProps = ViewProps<RoutineVersion>
