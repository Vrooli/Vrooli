import { StandardVersion, StandardVersionShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, FormProps, ObjectViewProps } from "../../../types";

type StandardUpsertPropsPage = CrudPropsPage;
type StandardUpsertPropsDialog = CrudPropsDialog<StandardVersion>;
export type StandardUpsertProps = StandardUpsertPropsPage | StandardUpsertPropsDialog;
export type StandardFormProps = FormProps<StandardVersion, StandardVersionShape> & {
    versions: string[];
}
export type StandardViewProps = ObjectViewProps<StandardVersion>
