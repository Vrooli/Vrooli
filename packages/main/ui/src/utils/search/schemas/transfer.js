import { TransferSortBy } from "@local/consts";
import { transferFindMany } from "../../../api/generated/endpoints/transfer_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const transferSearchSchema = () => ({
    formLayout: searchFormLayout("SearchTransfer"),
    containers: [],
    fields: [],
});
export const transferSearchParams = () => toParams(transferSearchSchema(), transferFindMany, TransferSortBy, TransferSortBy.DateCreatedDesc);
//# sourceMappingURL=transfer.js.map