import { PopularSortBy } from "@local/consts";
import { feedPopular } from "../../../api/generated/endpoints/feed_popular";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const popularSearchSchema = () => ({
    formLayout: searchFormLayout("SearchPopular"),
    containers: [],
    fields: [],
});
export const popularSearchParams = () => toParams(popularSearchSchema(), feedPopular, PopularSortBy, PopularSortBy.StarsDesc);
//# sourceMappingURL=popular.js.map