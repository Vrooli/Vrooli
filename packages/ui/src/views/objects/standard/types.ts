import { StandardVersion } from "@local/shared";
import { FormProps } from "forms/types";
import { StandardVersionShape } from "utils/shape/models/standardVersion";
import { ObjectViewProps } from "views/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type StandardUpsertPropsPage = CrudPropsPage;
type StandardUpsertPropsDialog = CrudPropsDialog<StandardVersion>;
export type StandardUpsertProps = StandardUpsertPropsPage | StandardUpsertPropsDialog;
export type StandardFormProps = FormProps<StandardVersion, StandardVersionShape> & {
    versions: string[];
}
export type StandardViewProps = ObjectViewProps<StandardVersion>
