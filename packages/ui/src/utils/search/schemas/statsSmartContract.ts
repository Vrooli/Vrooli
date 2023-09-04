import { endpointGetStatsSmartContract, StatsSmartContractSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsSmartContractSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchStatsSmartContract"),
    containers: [], //TODO
    fields: [], //TODO
});

export const statsSmartContractSearchParams = () => toParams(statsSmartContractSearchSchema(), endpointGetStatsSmartContract, undefined, StatsSmartContractSortBy, StatsSmartContractSortBy.PeriodStartAsc);
