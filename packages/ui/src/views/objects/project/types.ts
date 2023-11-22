import { ProjectVersion } from "@local/shared";
import { FormProps } from "forms/types";
import { ProjectVersionShape } from "utils/shape/models/projectVersion";
import { ObjectViewProps } from "views/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type ProjectUpsertPropsPage = CrudPropsPage;
type ProjectUpsertPropsDialog = CrudPropsDialog<ProjectVersion>;
export type ProjectUpsertProps = ProjectUpsertPropsPage | ProjectUpsertPropsDialog;
export type ProjectFormProps = FormProps<ProjectVersion, ProjectVersionShape> & {
    versions: string[];
}
export type ProjectViewProps = ObjectViewProps<ProjectVersion>
