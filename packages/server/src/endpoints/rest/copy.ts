import { copy_copy } from "../generated";
import { CopyEndpoints } from "../logic/copy";
import { setupRoutes } from "./base";

export const CopyRest = setupRoutes({
    "/copy": {
        post: [CopyEndpoints.Mutation.copy, copy_copy],
    },
});
