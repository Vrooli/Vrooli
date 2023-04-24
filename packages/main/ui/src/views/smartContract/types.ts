import { SmartContractVersion } from ":local/consts";
import { UpsertProps, ViewProps } from "../types";

export interface SmartContractUpsertProps extends UpsertProps<SmartContractVersion> { }
export interface SmartContractViewProps extends ViewProps<SmartContractVersion> { }
