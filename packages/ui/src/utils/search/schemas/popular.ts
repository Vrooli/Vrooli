import { PopularSortBy } from "@shared/consts";
import { feedPopular } from "api/generated/endpoints/feed_popular";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const popularSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchPopular'),
    containers: [], //TODO
    fields: [], //TODO
})

export const popularSearchParams = () => toParams(popularSearchSchema(), feedPopular, PopularSortBy, PopularSortBy.StarsDesc);