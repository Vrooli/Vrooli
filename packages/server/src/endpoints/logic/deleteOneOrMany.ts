import { Count, DeleteManyInput, DeleteOneInput, Success } from "@local/shared";
import { deleteManyHelper, deleteOneHelper } from "../../actions/deletes";
import { rateLimit } from "../../middleware/rateLimit";
import { GQLEndpoint } from "../../types";

export type EndpointsDeleteOneOrMany = {
    Mutation: {
        deleteOne: GQLEndpoint<DeleteOneInput, Success>;
        deleteMany: GQLEndpoint<DeleteManyInput, Count>;
    }
}

export const DeleteOneOrManyEndpoints: EndpointsDeleteOneOrMany = {
    Mutation: {
        deleteOne: async (_, { input }, { req }) => {
            await rateLimit({ maxUser: 1000, req });
            return deleteOneHelper({ input, req });
        },
        deleteMany: async (_, { input }, { req }) => {
            await rateLimit({ maxUser: 1000, req });
            return deleteManyHelper({ input, req });
        },
    },
};
