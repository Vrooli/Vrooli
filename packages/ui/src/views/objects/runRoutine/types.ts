import { RunRoutine } from "@local/shared";
import { FormProps } from "forms/types";
import { RunRoutineShape } from "utils/shape/models/runRoutine";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export type RunRoutineUpsertProps = UpsertProps<RunRoutine>
export type RunRoutineFormProps = FormProps<RunRoutine, RunRoutineShape>
export type RunRoutineViewProps = ObjectViewProps<RunRoutine>
