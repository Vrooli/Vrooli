import { endpointsFeed } from "@local/shared";
import { feed_home, popular_findMany } from "../generated";
import { FeedEndpoints } from "../logic/feed";
import { setupRoutes } from "./base";

export const FeedRest = setupRoutes([
    [endpointsFeed.home, FeedEndpoints.Query.home, feed_home],
    [endpointsFeed.popular, FeedEndpoints.Query.popular, popular_findMany],
]);
