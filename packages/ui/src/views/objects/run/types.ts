import { type Run, type RunShape } from "@local/shared";
import { type CrudPropsDialog, type CrudPropsPage, type CrudPropsPartial, type FormProps, type ObjectViewProps } from "../../../types.js";

type RunUpsertPropsPage = CrudPropsPage;
type RunUpsertPropsDialog = CrudPropsDialog<Run>;
type RunUpsertPropsPartial = CrudPropsPartial<Run>;
export type RunUpsertProps = RunUpsertPropsPage | RunUpsertPropsDialog | RunUpsertPropsPartial;
export type RunFormProps = FormProps<Run, RunShape>
export type RunViewProps = ObjectViewProps<Run>
