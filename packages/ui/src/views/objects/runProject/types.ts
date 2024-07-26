import { RunProject, RunProjectShape } from "@local/shared";
import { FormProps } from "forms/types";
import { ObjectViewProps } from "views/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type RunProjectUpsertPropsPage = CrudPropsPage;
type RunProjectUpsertPropsDialog = CrudPropsDialog<RunProject>;
export type RunProjectUpsertProps = RunProjectUpsertPropsPage | RunProjectUpsertPropsDialog;
export type RunProjectFormProps = FormProps<RunProject, RunProjectShape>
export type RunProjectViewProps = ObjectViewProps<RunProject>
