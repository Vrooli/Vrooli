import { ProjectVersion, ProjectVersionShape } from "@local/shared";
import { CrudProps, FormProps } from "../../../types";

export type ProjectCrudProps = CrudProps<ProjectVersion>;
export type ProjectFormProps = FormProps<ProjectVersion, ProjectVersionShape> & {
    versions: string[];
}
