import { BotCreateInput, BotUpdateInput, FindByIdOrHandleInput, ProfileEmailUpdateInput, ProfileUpdateInput, Success, User, UserDeleteInput, UserSearchInput } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { assertRequestFrom } from "../../auth";
import { CustomError } from "../../events";
import { rateLimit } from "../../middleware";
import { FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";
import { parseICalFile } from "../../utils";

export type EndpointsUser = {
    Query: {
        profile: GQLEndpoint<Record<string, never>, FindOneResult<User>>;
        user: GQLEndpoint<FindByIdOrHandleInput, FindOneResult<User>>;
        users: GQLEndpoint<UserSearchInput, FindManyResult<User>>;
    },
    Mutation: {
        botCreate: GQLEndpoint<BotCreateInput, UpdateOneResult<User>>;
        botUpdate: GQLEndpoint<BotUpdateInput, UpdateOneResult<User>>;
        profileUpdate: GQLEndpoint<ProfileUpdateInput, UpdateOneResult<User>>;
        profileEmailUpdate: GQLEndpoint<ProfileEmailUpdateInput, UpdateOneResult<User>>;
        userDeleteOne: GQLEndpoint<UserDeleteInput, Success>;
        importCalendar: GQLEndpoint<any, Success>;
        // importUserData: GQLEndpoint<ImportUserDataInput, Success>;
        exportCalendar: GQLEndpoint<Record<string, never>, string>;
        exportData: GQLEndpoint<Record<string, never>, string>;
    }
}

const objectType = "User";
export const UserEndpoints: EndpointsUser = {
    Query: {
        profile: async (_p, _d, { prisma, req }, info) => {
            const { id } = assertRequestFrom(req, { isUser: true });
            return readOneHelper({ info, input: { id }, objectType, prisma, req });
        },
        user: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        users: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        botCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        botUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        profileUpdate: async (_, { input }, { prisma, req, res }, info) => {
            await rateLimit({ maxUser: 250, req });
            // Add user id to input, since IDs are required for validation checks
            const { id } = assertRequestFrom(req, { isUser: true });
            return updateHelper({ info, input: { ...input, id }, objectType, prisma, req });
        },
        profileEmailUpdate: async (_, { input }, { prisma, req, res }, info) => {
            throw new CustomError("0999", "NotImplemented", ["en"]);
            // const userData = assertRequestFrom(req, { isUser: true });
            // await rateLimit({ maxUser: 100, req });
            // // Update object
            // const updated = await ProfileModel.mutate(prisma).updateEmails(userData.id, input, info);
            // if (!updated)
            //     throw new CustomError("0162", "ErrorUnknown", req.session.languages);
            // return updated;
        },
        userDeleteOne: async (_, { input }, { prisma, req, res }, info) => {
            throw new CustomError("0999", "NotImplemented", ["en"]);
            // const userData = assertRequestFrom(req, { isUser: true });
            // await rateLimit({ maxUser: 5, req });
            // // TODO anonymize public data
            // return await ProfileModel.mutate(prisma).deleteProfile(userData.id, input);
        },
        importCalendar: async (_, { input }, { prisma, req, res }, info) => {
            await rateLimit({ maxUser: 25, req });
            await parseICalFile(input.file);
            throw new CustomError("0999", "NotImplemented", ["en"]);
        },
        exportCalendar: async (_p, _d, { prisma, req, res }, info) => {
            throw new CustomError("0999", "NotImplemented", ["en"]);
        },
        /**
         * Exports user data to a JSON file (created/saved routines, projects, organizations, etc.).
         * @returns JSON of all user data
         */
        exportData: async (_p, _d, { prisma, req, res }, info) => {
            throw new CustomError("0999", "NotImplemented", ["en"]);
            // const userData = assertRequestFrom(req, { isUser: true });
            // await rateLimit({ maxUser: 5, req });
            // return await ProfileModel.port(prisma).exportData(userData.id);
        },
    },
};
