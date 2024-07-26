import { CodeVersion, CodeVersionShape } from "@local/shared";
import { FormProps } from "forms/types";
import { ObjectViewProps } from "views/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type SmartContractUpsertPropsPage = CrudPropsPage;
type SmartContractUpsertPropsDialog = CrudPropsDialog<CodeVersion>;
export type SmartContractUpsertProps = SmartContractUpsertPropsPage | SmartContractUpsertPropsDialog;
export type SmartContractFormProps = FormProps<CodeVersion, CodeVersionShape> & {
    versions: string[];
}
export type SmartContractViewProps = ObjectViewProps<CodeVersion>
