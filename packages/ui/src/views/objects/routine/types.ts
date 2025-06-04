import { type RoutineVersion, type RoutineVersionShape } from "@local/shared";
import { type CrudPropsDialog, type CrudPropsPage, type CrudPropsPartial, type FormProps, type ObjectViewProps } from "../../../types.js";

export type RoutineMultiStepCrudProps = CrudPropsPage;

type RoutineSingleStepUpsertPropsPage = CrudPropsPage & {
    isSubroutine?: boolean;
}
type RoutineSingleStepUpsertPropsDialog = CrudPropsDialog<RoutineVersion> & {
    isSubroutine?: boolean;
};
type RoutineSingleStepUpsertPropsPartial = CrudPropsPartial<RoutineVersion> & {
    isSubroutine?: boolean;
};
export type RoutineSingleStepUpsertProps = RoutineSingleStepUpsertPropsPage | RoutineSingleStepUpsertPropsDialog | RoutineSingleStepUpsertPropsPartial;
export type RoutineSingleStepFormProps = FormProps<RoutineVersion, RoutineVersionShape> & Pick<RoutineSingleStepUpsertProps, "isSubroutine"> & {
    versions: string[];
}
export type RoutineSingleStepViewProps = ObjectViewProps<RoutineVersion>
