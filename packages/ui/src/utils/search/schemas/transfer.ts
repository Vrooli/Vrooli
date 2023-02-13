import { TransferSortBy } from "@shared/consts";
import { transferFindMany } from "api/generated/endpoints/transfer";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const transferSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchTransfer', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const transferSearchParams = (lng: string) => toParams(transferSearchSchema(lng), transferFindMany, TransferSortBy, TransferSortBy.DateCreatedDesc);