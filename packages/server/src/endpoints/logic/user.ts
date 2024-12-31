import { AUTH_PROVIDERS, BotCreateInput, BotUpdateInput, DeleteType, FindByIdOrHandleInput, ImportCalendarInput, ProfileEmailUpdateInput, profileEmailUpdateValidation, ProfileUpdateInput, Session, Success, User, UserDeleteInput, UserSearchInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { cudHelper } from "../../actions/cuds";
import { deleteOneHelper } from "../../actions/deletes";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { PasswordAuthService } from "../../auth/email";
import { RequestService } from "../../auth/request";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { ApiEndpoint, FindManyResult, FindOneResult, IWrap, RecursivePartial, UpdateOneResult } from "../../types";
import { parseICalFile } from "../../utils";
import { AuthEndpoints } from "./auth";

export type EndpointsUser = {
    Query: {
        profile: ApiEndpoint<Record<string, never>, FindOneResult<User>>;
        user: ApiEndpoint<FindByIdOrHandleInput, FindOneResult<User>>;
        users: ApiEndpoint<UserSearchInput, FindManyResult<User>>;
    },
    Mutation: {
        botCreate: ApiEndpoint<BotCreateInput, UpdateOneResult<User>>;
        botUpdate: ApiEndpoint<BotUpdateInput, UpdateOneResult<User>>;
        profileUpdate: ApiEndpoint<ProfileUpdateInput, UpdateOneResult<User>>;
        profileEmailUpdate: ApiEndpoint<ProfileEmailUpdateInput, UpdateOneResult<User>>;
        userDeleteOne: ApiEndpoint<UserDeleteInput, RecursivePartial<Session>>;
        importCalendar: ApiEndpoint<ImportCalendarInput, Success>;
        // importUserData: ApiEndpoint<ImportUserDataInput, Success>;
        exportCalendar: ApiEndpoint<Record<string, never>, string>;
        exportData: ApiEndpoint<Record<string, never>, Success>;
    }
}

const objectType = "User";
export const UserEndpoints: EndpointsUser = {
    Query: {
        profile: async (_p, _d, { req }, info) => {
            const { id } = RequestService.assertRequestFrom(req, { isUser: true });
            return readOneHelper({ info, input: { id }, objectType, req });
        },
        user: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        users: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        botCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 500, req });
            return createOneHelper({ info, input, objectType, req });
        },
        botUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return updateOneHelper({ info, input, objectType, req });
        },
        profileUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            // Add user id to input, since IDs are required for validation checks
            const { id } = RequestService.assertRequestFrom(req, { isUser: true });
            return updateOneHelper({ info, input: { ...input, id }, objectType, req });
        },
        profileEmailUpdate: async (_, { input }, { req }, info) => {
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 100, req });
            // Validate input
            profileEmailUpdateValidation.update({ env: process.env.NODE_ENV as "development" | "production" }).validateSync(input, { abortEarly: false });
            // Find user
            const user = await prismaInstance.user.findUnique({
                where: { id: userData.id },
                select: PasswordAuthService.selectUserForPasswordAuth(),
            });
            if (!user)
                throw new CustomError("0062", "InvalidCredentials"); // Purposefully vague with duplicate code for security
            // If user doesn't have a password, they must reset it first
            const passwordAuth = user.auths.find((auth) => auth.provider === AUTH_PROVIDERS.Password);
            const passwordHash = passwordAuth?.hashed_password ?? null;
            if (!passwordAuth || !passwordHash) {
                await PasswordAuthService.setupPasswordReset(user);
                throw new CustomError("0125", "MustResetPassword");
            }
            // If new password is provided
            if (input.newPassword) {
                // Validate current password
                const session = await PasswordAuthService.logIn(input.currentPassword, user, req);
                if (!session) {
                    throw new CustomError("0062", "InvalidCredentials"); // Purposefully vague with duplicate code for security
                }
                // Update password
                await prismaInstance.user_auth.update({
                    where: {
                        id: passwordAuth.id,
                    },
                    data: {
                        hashed_password: PasswordAuthService.hashPassword(input.newPassword),
                    },
                });
            }
            // Create new emails
            if (input.emailsCreate) {
                await cudHelper({
                    info: { __typename: "Email", id: true, emailAddress: true },
                    inputData: input.emailsCreate.map(email => ({ action: "Create", input: email, objectType: "Email" })),
                    userData,
                });
            }
            // Delete emails
            if (input.emailsDelete) {
                // Make sure you have at least one authentication method remaining
                const emailsCount = await prismaInstance.email.count({ where: { userId: userData.id } });
                const walletsCount = await prismaInstance.wallet.count({ where: { userId: userData.id } });
                if (emailsCount - input.emailsDelete.length <= 0 && walletsCount <= 0)
                    throw new CustomError("0126", "MustLeaveVerificationMethod");
                // Delete emails
                await prismaInstance.email.deleteMany({ where: { id: { in: input.emailsDelete } } });
            }
            // Return updated user
            return readOneHelper({ info, input: { id: userData.id }, objectType, req });
        },
        userDeleteOne: async (_, { input }, { req, res }, info) => {
            const { id } = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 500, req });
            // Find user
            const user = await prismaInstance.user.findUnique({
                where: { id },
                select: PasswordAuthService.selectUserForPasswordAuth(),
            });
            if (!user)
                throw new CustomError("0062", "InvalidCredentials"); // Purposefully vague with duplicate code for security
            // If user doesn't have a password, they must reset it first
            const passwordHash = PasswordAuthService.getAuthPassword(user);
            if (!passwordHash) {
                await PasswordAuthService.setupPasswordReset(user);
                throw new CustomError("0246", "MustResetPassword");
            }
            // Use the log in logic to check password
            const session = await PasswordAuthService.logIn(input?.password as string, user, req);
            if (!session) {
                throw new CustomError("0062", "InvalidCredentials"); // Purposefully vague with duplicate code for security
            }
            // TODO anonymize public data
            // Delete user
            const result = await deleteOneHelper({ input: { id, objectType: objectType as DeleteType }, req });
            // If successful, remove user from session
            if (result.success) {
                // TODO this will only work if you're deleing yourself, since it clears your own session. 
                // If there are cases like deleting a bot, or an admin deleting a user, this will need to be handled differently
                return AuthEndpoints.Mutation.logOut(undefined, undefined as unknown as IWrap<Record<string, never>>, { req, res }, info) as any;
            } else {
                throw new CustomError("0123", "InternalError");
            }
        },
        importCalendar: async (_, { input }, { req }) => {
            await RequestService.get().rateLimit({ maxUser: 25, req });
            await parseICalFile(input.file);
            throw new CustomError("0999", "NotImplemented");
        },
        exportCalendar: async (_p, _d) => {
            throw new CustomError("0999", "NotImplemented");
        },
        /**
         * Exports user data to a JSON file (created/saved routines, projects, teams, etc.).
         * @returns JSON of all user data
         */
        exportData: async (_p, _d) => {
            throw new CustomError("0999", "NotImplemented");
            // const userData = RequestService.assertRequestFrom(req, { isUser: true });
            // await RequestService.get().rateLimit({ maxUser: 5, req });
            // return await ProfileModel.port().exportData(userData.id);
        },
    },
};
