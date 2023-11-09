import { SmartContractVersion } from "@local/shared";
import { FormProps } from "forms/types";
import { SmartContractVersionShape } from "utils/shape/models/smartContractVersion";
import { ObjectViewProps } from "views/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type SmartContractUpsertPropsPage = CrudPropsPage;
type SmartContractUpsertPropsDialog = CrudPropsDialog<SmartContractVersion>;
export type SmartContractUpsertProps = SmartContractUpsertPropsPage | SmartContractUpsertPropsDialog;
export type SmartContractFormProps = FormProps<SmartContractVersion, SmartContractVersionShape> & {
    versions: string[];
}
export type SmartContractViewProps = ObjectViewProps<SmartContractVersion>
