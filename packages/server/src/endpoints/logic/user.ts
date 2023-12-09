import { BotCreateInput, BotUpdateInput, FindByIdOrHandleInput, ImportCalendarInput, ProfileEmailUpdateInput, profileEmailUpdateValidation, ProfileUpdateInput, Session, Success, User, UserDeleteInput, UserSearchInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { cudHelper } from "../../actions/cuds";
import { deleteOneHelper } from "../../actions/deletes";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { hashPassword, logIn, setupPasswordReset } from "../../auth/email";
import { assertRequestFrom } from "../../auth/request";
import { CustomError } from "../../events/error";
import { rateLimit } from "../../middleware/rateLimit";
import { FindManyResult, FindOneResult, GQLEndpoint, RecursivePartial, UpdateOneResult } from "../../types";
import { parseICalFile } from "../../utils";
import { AuthEndpoints } from "./auth";

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
        userDeleteOne: GQLEndpoint<UserDeleteInput, RecursivePartial<Session>>;
        importCalendar: GQLEndpoint<ImportCalendarInput, Success>;
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
            return createOneHelper({ info, input, objectType, prisma, req });
        },
        botUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return updateOneHelper({ info, input, objectType, prisma, req });
        },
        profileUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            // Add user id to input, since IDs are required for validation checks
            const { id } = assertRequestFrom(req, { isUser: true });
            return updateOneHelper({ info, input: { ...input, id }, objectType, prisma, req });
        },
        profileEmailUpdate: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 100, req });
            // Validate input
            profileEmailUpdateValidation.update({ env: process.env.NODE_ENV as "development" | "production" }).validateSync(input, { abortEarly: false });
            // Find user
            const user = await prisma.user.findUnique({ where: { id: userData.id } });
            if (!user)
                throw new CustomError("0124", "NoUser", req.session.languages);
            // If user doesn't have a password, they must reset it first
            if (!user.password) {
                await setupPasswordReset(user, prisma);
                throw new CustomError("0125", "MustResetPassword", req.session.languages);
            }
            // If new password is provided
            if (input.newPassword) {
                // Validate current password
                const session = await logIn(input.currentPassword, user, prisma, req);
                if (!session) {
                    throw new CustomError("0127", "BadCredentials", req.session.languages);
                }
                // Update password
                await prisma.user.update({ where: { id: userData.id }, data: { password: hashPassword(input.newPassword) } });
            }
            // Create new emails
            if (input.emailsCreate) {
                await cudHelper({
                    inputData: input.emailsCreate.map(email => ({ actionType: "Create", input: email, objectType: "Email" })),
                    partialInfo: { __typename: "Email", id: true, emailAddress: true },
                    prisma,
                    userData,
                });
            }
            // Delete emails
            if (input.emailsDelete) {
                // Make sure you have at least one authentication method remaining
                const emailsCount = await prisma.email.count({ where: { userId: userData.id } });
                const walletsCount = await prisma.wallet.count({ where: { userId: userData.id } });
                if (emailsCount - input.emailsDelete.length <= 0 && walletsCount <= 0)
                    throw new CustomError("0126", "MustLeaveVerificationMethod", req.session.languages);
                // Delete emails
                await prisma.email.deleteMany({ where: { id: { in: input.emailsDelete } } });
            }
            // Return updated user
            return readOneHelper({ info, input: { id: userData.id }, objectType, prisma, req });
        },
        userDeleteOne: async (_, { input }, { prisma, req, res }, info) => {
            const { id } = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 500, req });
            // Find user
            const user = await prisma.user.findUnique({ where: { id } });
            if (!user)
                throw new CustomError("0245", "NoUser", req.session.languages);
            // If user doesn't have a password, they must reset it first
            if (!user.password) {
                await setupPasswordReset(user, prisma);
                throw new CustomError("0246", "MustResetPassword", req.session.languages);
            }
            // Use the log in logic to check password
            const session = await logIn(input?.password as string, user, prisma, req);
            if (!session) {
                throw new CustomError("0048", "BadCredentials", req.session.languages);
            }
            // TODO anonymize public data
            // Delete user
            const result = await deleteOneHelper({ input: { id }, objectType, prisma, req });
            // If successful, remove user from session
            if (result.success) {
                return AuthEndpoints.Mutation.logOut(undefined, { input: { id } }, { prisma, req, res }, info) as any;
            } else {
                throw new CustomError("0123", "InternalError", req.session.languages);
            }
        },
        importCalendar: async (_, { input }, { req }) => {
            await rateLimit({ maxUser: 25, req });
            await parseICalFile(input.file);
            throw new CustomError("0999", "NotImplemented", ["en"]);
        },
        exportCalendar: async (_p, _d, { prisma }) => {
            throw new CustomError("0999", "NotImplemented", ["en"]);
        },
        /**
         * Exports user data to a JSON file (created/saved routines, projects, organizations, etc.).
         * @returns JSON of all user data
         */
        exportData: async (_p, _d, { prisma }) => {
            throw new CustomError("0999", "NotImplemented", ["en"]);
            // const userData = assertRequestFrom(req, { isUser: true });
            // await rateLimit({ maxUser: 5, req });
            // return await ProfileModel.port(prisma).exportData(userData.id);
        },
    },
};
