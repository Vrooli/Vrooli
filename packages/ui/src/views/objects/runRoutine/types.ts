import { RunRoutine } from "@local/shared";
import { FormProps } from "forms/types";
import { RunRoutineShape } from "utils/shape/models/runRoutine";
import { ObjectViewProps } from "views/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type RunRoutineUpsertPropsPage = CrudPropsPage;
type RunRoutineUpsertPropsDialog = CrudPropsDialog<RunRoutine>;
export type RunRoutineUpsertProps = RunRoutineUpsertPropsPage | RunRoutineUpsertPropsDialog;
export type RunRoutineFormProps = FormProps<RunRoutine, RunRoutineShape>
export type RunRoutineViewProps = ObjectViewProps<RunRoutine>
