import { RunRoutine, RunRoutineShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, FormProps, ObjectViewProps } from "../../../types.js";

type RunRoutineUpsertPropsPage = CrudPropsPage;
type RunRoutineUpsertPropsDialog = CrudPropsDialog<RunRoutine>;
export type RunRoutineUpsertProps = RunRoutineUpsertPropsPage | RunRoutineUpsertPropsDialog;
export type RunRoutineFormProps = FormProps<RunRoutine, RunRoutineShape>
export type RunRoutineViewProps = ObjectViewProps<RunRoutine>
