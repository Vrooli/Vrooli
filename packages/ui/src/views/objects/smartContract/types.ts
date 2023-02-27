import { SmartContractVersion } from "@shared/consts";
import { CreateProps, UpdateProps, ViewProps } from "../types";

export interface SmartContractCreateProps extends CreateProps<SmartContractVersion> {}
export interface SmartContractUpdateProps extends UpdateProps<SmartContractVersion> {}
export interface SmartContractViewProps extends ViewProps<SmartContractVersion> {}