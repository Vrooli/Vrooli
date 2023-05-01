import { TransferSortBy } from "@local/shared";
import { transferFindMany } from "api/generated/endpoints/transfer_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const transferSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchTransfer"),
    containers: [], //TODO
    fields: [], //TODO
});

export const transferSearchParams = () => toParams(transferSearchSchema(), transferFindMany, TransferSortBy, TransferSortBy.DateCreatedDesc);
