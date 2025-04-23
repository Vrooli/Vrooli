import { CodeVersion, CodeVersionShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, CrudPropsPartial, FormProps, ObjectViewProps } from "../../../types.js";

type SmartContractUpsertPropsPage = CrudPropsPage;
type SmartContractUpsertPropsDialog = CrudPropsDialog<CodeVersion>;
type SmartContractUpsertPropsPartial = CrudPropsPartial<CodeVersion>;
export type SmartContractUpsertProps = SmartContractUpsertPropsPage | SmartContractUpsertPropsDialog | SmartContractUpsertPropsPartial;
export type SmartContractFormProps = FormProps<CodeVersion, CodeVersionShape> & {
    versions: string[];
}
export type SmartContractViewProps = ObjectViewProps<CodeVersion>
