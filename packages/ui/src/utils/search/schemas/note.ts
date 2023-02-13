import { NoteSortBy } from "@shared/consts";
import { noteFindMany } from "api/generated/endpoints/note";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const noteSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchNote', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const noteSearchParams = (lng: string) => toParams(noteSearchSchema(lng), noteFindMany, NoteSortBy, NoteSortBy.ScoreDesc);