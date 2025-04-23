import { RunRoutine, RunRoutineShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, CrudPropsPartial, FormProps, ObjectViewProps } from "../../../types.js";

type RunRoutineUpsertPropsPage = CrudPropsPage;
type RunRoutineUpsertPropsDialog = CrudPropsDialog<RunRoutine>;
type RunRoutineUpsertPropsPartial = CrudPropsPartial<RunRoutine>;
export type RunRoutineUpsertProps = RunRoutineUpsertPropsPage | RunRoutineUpsertPropsDialog | RunRoutineUpsertPropsPartial;
export type RunRoutineFormProps = FormProps<RunRoutine, RunRoutineShape>
export type RunRoutineViewProps = ObjectViewProps<RunRoutine>
