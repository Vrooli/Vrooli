import { feed_home, feed_popular } from "@local/shared";
import { FeedEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const FeedRest = setupRoutes({
    "/feed/home": {
        get: [FeedEndpoints.Query.home, feed_home],
    },
    "/feed/popular": {
        get: [FeedEndpoints.Query.popular, feed_popular],
    },
});
