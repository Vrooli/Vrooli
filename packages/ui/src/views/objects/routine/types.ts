import { RoutineVersion } from "@local/shared";
import { FormProps } from "forms/types";
import { RoutineVersionShape } from "utils/shape/models/routineVersion";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export type RoutineUpsertProps = UpsertProps<RoutineVersion> & {
    isSubroutine?: boolean;
}
export type RoutineFormProps = FormProps<RoutineVersion, RoutineVersionShape> & Pick<RoutineUpsertProps, "isSubroutine"> & {
    versions: string[];
}
export type RoutineViewProps = ObjectViewProps<RoutineVersion>
