import { deleteOneOrMany_deleteMany, deleteOneOrMany_deleteOne } from "@local/shared";
import { DeleteOneOrManyEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const DeleteOneOrManyRest = setupRoutes({
    "/deleteOne": {
        post: [DeleteOneOrManyEndpoints.Mutation.deleteOne, deleteOneOrMany_deleteOne],
    },
    "/deleteMany": {
        post: [DeleteOneOrManyEndpoints.Mutation.deleteMany, deleteOneOrMany_deleteMany],
    },
});
