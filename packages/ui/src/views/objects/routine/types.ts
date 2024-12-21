import { RoutineVersion, RoutineVersionShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, FormProps, ObjectViewProps } from "../../../types";

export type RoutineMultiStepCrudProps = CrudPropsPage;

type RoutineSingleStepUpsertPropsPage = CrudPropsPage & {
    isSubroutine?: boolean;
}
type RoutineSingleStepUpsertPropsDialog = CrudPropsDialog<RoutineVersion> & {
    isSubroutine?: boolean;
};
export type RoutineSingleStepUpsertProps = RoutineSingleStepUpsertPropsPage | RoutineSingleStepUpsertPropsDialog;
export type RoutineSingleStepFormProps = FormProps<RoutineVersion, RoutineVersionShape> & Pick<RoutineSingleStepUpsertProps, "isSubroutine"> & {
    versions: string[];
}
export type RoutineSingleStepViewProps = ObjectViewProps<RoutineVersion>
