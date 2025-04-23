import { RunProject, RunProjectShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, CrudPropsPartial, FormProps, ObjectViewProps } from "../../../types.js";

type RunProjectUpsertPropsPage = CrudPropsPage;
type RunProjectUpsertPropsDialog = CrudPropsDialog<RunProject>;
type RunProjectUpsertPropsPartial = CrudPropsPartial<RunProject>;
export type RunProjectUpsertProps = RunProjectUpsertPropsPage | RunProjectUpsertPropsDialog | RunProjectUpsertPropsPartial;
export type RunProjectFormProps = FormProps<RunProject, RunProjectShape>
export type RunProjectViewProps = ObjectViewProps<RunProject>
