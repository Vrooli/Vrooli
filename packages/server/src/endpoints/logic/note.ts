import { FindByIdInput, Note, NoteCreateInput, NoteSearchInput, NoteUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsNote = {
    Query: {
        note: GQLEndpoint<FindByIdInput, FindOneResult<Note>>;
        notes: GQLEndpoint<NoteSearchInput, FindManyResult<Note>>;
    },
    Mutation: {
        noteCreate: GQLEndpoint<NoteCreateInput, CreateOneResult<Note>>;
        noteUpdate: GQLEndpoint<NoteUpdateInput, UpdateOneResult<Note>>;
    }
}

const objectType = "Note";
export const NoteEndpoints: EndpointsNote = {
    Query: {
        note: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        notes: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        noteCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        noteUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
