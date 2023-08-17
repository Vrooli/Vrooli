import { RoutineVersion } from "@local/shared";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export interface RoutineUpsertProps extends UpsertProps<RoutineVersion> {
    isSubroutine?: boolean;
}
export type RoutineViewProps = ObjectViewProps<RoutineVersion>
