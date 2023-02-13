import { ApiSortBy } from "@shared/consts";
import { apiFindMany } from "api/generated/endpoints/api";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const apiSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchApi', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const apiSearchParams = (lng: string) => toParams(apiSearchSchema(lng), apiFindMany, ApiSortBy, ApiSortBy.ScoreDesc);