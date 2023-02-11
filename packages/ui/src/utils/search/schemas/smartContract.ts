import { SmartContractSortBy } from "@shared/consts";
import { smartContractFindMany } from "api/generated/endpoints/smartContract";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const smartContractSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchSmartContracts', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const smartContractSearchParams = (lng: string) => toParams(smartContractSearchSchema(lng), smartContractFindMany, SmartContractSortBy, SmartContractSortBy.ScoreDesc);