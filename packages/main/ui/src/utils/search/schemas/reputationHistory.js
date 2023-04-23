import { ReputationHistorySortBy } from "@local/consts";
import { reputationHistoryFindMany } from "../../../api/generated/endpoints/reputationHistory_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const reputationHistorySearchSchema = () => ({
    formLayout: searchFormLayout("SearchReputationHistory"),
    containers: [],
    fields: [],
});
export const reputationHistorySearchParams = () => toParams(reputationHistorySearchSchema(), reputationHistoryFindMany, ReputationHistorySortBy, ReputationHistorySortBy.DateCreatedDesc);
//# sourceMappingURL=reputationHistory.js.map