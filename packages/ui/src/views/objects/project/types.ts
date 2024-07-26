import { ProjectVersion, ProjectVersionShape } from "@local/shared";
import { FormProps } from "forms/types";
import { CrudProps } from "../types";

export type ProjectCrudProps = CrudProps<ProjectVersion>;
export type ProjectFormProps = FormProps<ProjectVersion, ProjectVersionShape> & {
    versions: string[];
}
