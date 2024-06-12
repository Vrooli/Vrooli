import { ProjectVersion } from "@local/shared";
import { FormProps } from "forms/types";
import { ProjectVersionShape } from "utils/shape/models/projectVersion";
import { CrudProps } from "../types";

export type ProjectCrudProps = CrudProps<ProjectVersion>;
export type ProjectFormProps = FormProps<ProjectVersion, ProjectVersionShape> & {
    versions: string[];
}
