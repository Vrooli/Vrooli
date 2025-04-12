import { CodeVersion, CodeVersionShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, FormProps, ObjectViewProps } from "../../../types.js";

type SmartContractUpsertPropsPage = CrudPropsPage;
type SmartContractUpsertPropsDialog = CrudPropsDialog<CodeVersion>;
export type SmartContractUpsertProps = SmartContractUpsertPropsPage | SmartContractUpsertPropsDialog;
export type SmartContractFormProps = FormProps<CodeVersion, CodeVersionShape> & {
    versions: string[];
}
export type SmartContractViewProps = ObjectViewProps<CodeVersion>
