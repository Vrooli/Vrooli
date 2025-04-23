import { StandardVersion, StandardVersionShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, CrudPropsPartial, FormProps, ObjectViewProps } from "../../../types.js";

type DataStructureUpsertPropsPage = CrudPropsPage;
type DataStructureUpsertPropsDialog = CrudPropsDialog<StandardVersion>;
type DataStructureUpsertPropsPartial = CrudPropsPartial<StandardVersion>;
export type DataStructureUpsertProps = DataStructureUpsertPropsPage | DataStructureUpsertPropsDialog | DataStructureUpsertPropsPartial;
export type DataStructureFormProps = FormProps<StandardVersion, StandardVersionShape> & {
    versions: string[];
}
export type DataStructureViewProps = ObjectViewProps<StandardVersion>
