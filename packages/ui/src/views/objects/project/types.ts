import { type ProjectVersion, type ProjectVersionShape } from "@local/shared";
import { type CrudProps, type FormProps } from "../../../types.js";

export type ProjectCrudProps = CrudProps<ProjectVersion>;
export type ProjectFormProps = FormProps<ProjectVersion, ProjectVersionShape> & {
    versions: string[];
}
