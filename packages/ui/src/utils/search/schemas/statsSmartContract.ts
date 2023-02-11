import { StatsSmartContractSortBy } from "@shared/consts";
import { statsSmartContractFindMany } from "api/generated/endpoints/statsSmartContract";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsSmartContractSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchStatsSmartContract', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsSmartContractSearchParams = (lng: string) => toParams(statsSmartContractSearchSchema(lng), statsSmartContractFindMany, StatsSmartContractSortBy, StatsSmartContractSortBy.DateUpdatedDesc);