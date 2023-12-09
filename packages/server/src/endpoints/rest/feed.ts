import { feed_home, popular_findMany } from "../generated";
import { FeedEndpoints } from "../logic/feed";
import { setupRoutes } from "./base";

export const FeedRest = setupRoutes({
    "/feed/home": {
        get: [FeedEndpoints.Query.home, feed_home],
    },
    "/feed/popular": {
        get: [FeedEndpoints.Query.popular, popular_findMany],
    },
});
