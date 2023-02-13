import { SmartContractVersionSortBy } from "@shared/consts";
import { smartContractVersionFindMany } from "api/generated/endpoints/smartContractVersion";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const smartContractVersionSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchSmartContractVersion', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const smartContractVersionSearchParams = (lng: string) => toParams(smartContractVersionSearchSchema(lng), smartContractVersionFindMany, SmartContractVersionSortBy, SmartContractVersionSortBy.DateCreatedDesc);