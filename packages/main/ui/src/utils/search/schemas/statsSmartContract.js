import { StatsSmartContractSortBy } from "@local/consts";
import { statsSmartContractFindMany } from "../../../api/generated/endpoints/statsSmartContract_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const statsSmartContractSearchSchema = () => ({
    formLayout: searchFormLayout("SearchStatsSmartContract"),
    containers: [],
    fields: [],
});
export const statsSmartContractSearchParams = () => toParams(statsSmartContractSearchSchema(), statsSmartContractFindMany, StatsSmartContractSortBy, StatsSmartContractSortBy.PeriodStartAsc);
//# sourceMappingURL=statsSmartContract.js.map