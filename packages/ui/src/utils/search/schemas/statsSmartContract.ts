import { statsSmartContractFindMany, StatsSmartContractSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsSmartContractSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchStatsSmartContract"),
    containers: [], //TODO
    fields: [], //TODO
});

export const statsSmartContractSearchParams = () => toParams(statsSmartContractSearchSchema(), statsSmartContractFindMany, StatsSmartContractSortBy, StatsSmartContractSortBy.PeriodStartAsc);
