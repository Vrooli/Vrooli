import { FindByIdInput, Note, NoteCreateInput, NoteSearchInput, NoteUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsNote = {
    Query: {
        note: ApiEndpoint<FindByIdInput, FindOneResult<Note>>;
        notes: ApiEndpoint<NoteSearchInput, FindManyResult<Note>>;
    },
    Mutation: {
        noteCreate: ApiEndpoint<NoteCreateInput, CreateOneResult<Note>>;
        noteUpdate: ApiEndpoint<NoteUpdateInput, UpdateOneResult<Note>>;
    }
}

const objectType = "Note";
export const NoteEndpoints: EndpointsNote = {
    Query: {
        note: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        notes: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        noteCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        noteUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
