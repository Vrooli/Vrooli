import { SmartContractVersion } from "@local/shared";
import { UpsertProps, ViewProps } from "../types";

export interface SmartContractUpsertProps extends UpsertProps<SmartContractVersion> { }
export interface SmartContractViewProps extends ViewProps<SmartContractVersion> { }