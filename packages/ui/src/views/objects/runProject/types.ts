import { RunProject } from "@local/shared";
import { FormProps } from "forms/types";
import { RunProjectShape } from "utils/shape/models/runProject";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export type RunProjectUpsertProps = UpsertProps<RunProject>
export type RunProjectFormProps = FormProps<RunProject, RunProjectShape>
export type RunProjectViewProps = ObjectViewProps<RunProject>
