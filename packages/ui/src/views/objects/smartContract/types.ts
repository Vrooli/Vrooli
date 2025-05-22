import { type CodeVersion, type CodeVersionShape } from "@local/shared";
import { type CrudPropsDialog, type CrudPropsPage, type CrudPropsPartial, type FormProps, type ObjectViewProps } from "../../../types.js";

type SmartContractUpsertPropsPage = CrudPropsPage;
type SmartContractUpsertPropsDialog = CrudPropsDialog<CodeVersion>;
type SmartContractUpsertPropsPartial = CrudPropsPartial<CodeVersion>;
export type SmartContractUpsertProps = SmartContractUpsertPropsPage | SmartContractUpsertPropsDialog | SmartContractUpsertPropsPartial;
export type SmartContractFormProps = FormProps<CodeVersion, CodeVersionShape> & {
    versions: string[];
}
export type SmartContractViewProps = ObjectViewProps<CodeVersion>
