import { StandardVersion, StandardVersionShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, FormProps, ObjectViewProps } from "../../../types.js";

type DataStructureUpsertPropsPage = CrudPropsPage;
type DataStructureUpsertPropsDialog = CrudPropsDialog<StandardVersion>;
export type DataStructureUpsertProps = DataStructureUpsertPropsPage | DataStructureUpsertPropsDialog;
export type DataStructureFormProps = FormProps<StandardVersion, StandardVersionShape> & {
    versions: string[];
}
export type DataStructureViewProps = ObjectViewProps<StandardVersion>
