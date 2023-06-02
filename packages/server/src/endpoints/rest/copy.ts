import { copy_copy } from "@local/shared";
import { CopyEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const CopyRest = setupRoutes({
    "/copy": {
        post: [CopyEndpoints.Mutation.copy, copy_copy],
    },
});
