import { Run, RunShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, CrudPropsPartial, FormProps, ObjectViewProps } from "../../../types.js";

type RunUpsertPropsPage = CrudPropsPage;
type RunUpsertPropsDialog = CrudPropsDialog<Run>;
type RunUpsertPropsPartial = CrudPropsPartial<Run>;
export type RunUpsertProps = RunUpsertPropsPage | RunUpsertPropsDialog | RunUpsertPropsPartial;
export type RunFormProps = FormProps<Run, RunShape>
export type RunViewProps = ObjectViewProps<Run>
