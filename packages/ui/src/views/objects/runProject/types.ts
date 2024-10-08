import { RunProject, RunProjectShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, FormProps, ObjectViewProps } from "../../../types";

type RunProjectUpsertPropsPage = CrudPropsPage;
type RunProjectUpsertPropsDialog = CrudPropsDialog<RunProject>;
export type RunProjectUpsertProps = RunProjectUpsertPropsPage | RunProjectUpsertPropsDialog;
export type RunProjectFormProps = FormProps<RunProject, RunProjectShape>
export type RunProjectViewProps = ObjectViewProps<RunProject>
