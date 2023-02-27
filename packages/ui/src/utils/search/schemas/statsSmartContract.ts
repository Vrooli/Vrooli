import { StatsSmartContractSortBy } from "@shared/consts";
import { statsSmartContractFindMany } from "api/generated/endpoints/statsSmartContract";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsSmartContractSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchStatsSmartContract'),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsSmartContractSearchParams = () => toParams(statsSmartContractSearchSchema(), statsSmartContractFindMany, StatsSmartContractSortBy, StatsSmartContractSortBy.DateUpdatedDesc);