import { SmartContractVersion } from "@local/shared";
import { FormProps } from "forms/types";
import { SmartContractVersionShape } from "utils/shape/models/smartContractVersion";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export type SmartContractUpsertProps = UpsertProps<SmartContractVersion>
export type SmartContractFormProps = FormProps<SmartContractVersion, SmartContractVersionShape> & {
    versions: string[];
}
export type SmartContractViewProps = ObjectViewProps<SmartContractVersion>
