import { RoutineVersion } from "@local/shared";
import { FormProps } from "forms/types";
import { RoutineVersionShape } from "utils/shape/models/routineVersion";
import { ObjectViewProps } from "views/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type RoutineUpsertPropsPage = CrudPropsPage & {
    isSubroutine?: boolean;
}
type RoutineUpsertPropsDialog = CrudPropsDialog<RoutineVersion> & {
    isSubroutine?: boolean;
};
export type RoutineUpsertProps = RoutineUpsertPropsPage | RoutineUpsertPropsDialog;
export type RoutineFormProps = FormProps<RoutineVersion, RoutineVersionShape> & Pick<RoutineUpsertProps, "isSubroutine"> & {
    versions: string[];
}
export type RoutineViewProps = ObjectViewProps<RoutineVersion>
