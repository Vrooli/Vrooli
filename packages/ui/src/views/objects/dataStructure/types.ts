import { type StandardVersion, type StandardVersionShape } from "@local/shared";
import { type CrudPropsDialog, type CrudPropsPage, type CrudPropsPartial, type FormProps, type ObjectViewProps } from "../../../types.js";

type DataStructureUpsertPropsPage = CrudPropsPage;
type DataStructureUpsertPropsDialog = CrudPropsDialog<StandardVersion>;
type DataStructureUpsertPropsPartial = CrudPropsPartial<StandardVersion>;
export type DataStructureUpsertProps = DataStructureUpsertPropsPage | DataStructureUpsertPropsDialog | DataStructureUpsertPropsPartial;
export type DataStructureFormProps = FormProps<StandardVersion, StandardVersionShape> & {
    versions: string[];
}
export type DataStructureViewProps = ObjectViewProps<StandardVersion>
