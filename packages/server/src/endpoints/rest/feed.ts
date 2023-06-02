import { feed_home, feed_popular } from "@local/shared";
import { FeedEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const FeedRest = setupRoutes({
    "/feed/home": {
        post: [FeedEndpoints.Query.home, feed_home],
    },
    "/feed/popular": {
        post: [FeedEndpoints.Query.popular, feed_popular],
    },
});
