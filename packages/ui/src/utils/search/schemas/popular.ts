import { PopularSortBy } from "@shared/consts";
import { FormSchema } from "../../../forms/types";
import { feedPopular } from "../../../api/generated/endpoints/feed_popular";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const popularSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchPopular"),
    containers: [], //TODO
    fields: [], //TODO
})

export const popularSearchParams = () => toParams(popularSearchSchema(), feedPopular, PopularSortBy, PopularSortBy.StarsDesc);