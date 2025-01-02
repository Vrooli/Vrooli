import { FindVersionInput, NoteVersion, NoteVersionCreateInput, NoteVersionSearchInput, NoteVersionUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsNoteVersion = {
    findOne: ApiEndpoint<FindVersionInput, FindOneResult<NoteVersion>>;
    findMany: ApiEndpoint<NoteVersionSearchInput, FindManyResult<NoteVersion>>;
    createOne: ApiEndpoint<NoteVersionCreateInput, CreateOneResult<NoteVersion>>;
    updateOne: ApiEndpoint<NoteVersionUpdateInput, UpdateOneResult<NoteVersion>>;
}

const objectType = "NoteVersion";
export const noteVersion: EndpointsNoteVersion = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    createOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
