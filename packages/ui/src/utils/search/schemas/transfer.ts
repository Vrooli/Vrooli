import { endpointGetTransfer, endpointGetTransfers, TransferSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const transferSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchTransfer"),
    containers: [], //TODO
    fields: [], //TODO
});

export const transferSearchParams = () => toParams(transferSearchSchema(), endpointGetTransfers, endpointGetTransfer, TransferSortBy, TransferSortBy.DateCreatedDesc);
