import { ProjectVersion } from "@local/shared";
import { FormProps } from "forms/types";
import { ProjectVersionShape } from "utils/shape/models/projectVersion";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export type ProjectUpsertProps = UpsertProps<ProjectVersion>
export type ProjectFormProps = FormProps<ProjectVersion, ProjectVersionShape> & {
    versions: string[];
}
export type ProjectViewProps = ObjectViewProps<ProjectVersion>
