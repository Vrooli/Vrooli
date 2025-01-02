import { CopyInput, CopyResult, Count, DeleteAccountInput, DeleteAllInput, DeleteManyInput, DeleteOneInput, DeleteType, Success, lowercaseFirstLetter } from "@local/shared";
import { copyHelper } from "../../actions/copies";
import { deleteManyHelper, deleteOneHelper } from "../../actions/deletes";
import { PasswordAuthService } from "../../auth/email";
import { RequestService } from "../../auth/request";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { ModelMap } from "../../models/base";
import { ApiEndpoint, IWrap } from "../../types";
import { auth } from "./auth";

export type EndpointsActions = {
    copy: ApiEndpoint<CopyInput, CopyResult>;
    deleteOne: ApiEndpoint<DeleteOneInput, Success>;
    deleteMany: ApiEndpoint<DeleteManyInput, Count>;
    deleteAll: ApiEndpoint<DeleteAllInput, Count>;
    deleteAccount: ApiEndpoint<DeleteAccountInput, Success>;
}

export const actions: EndpointsActions = {
    copy: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        const result = await copyHelper({ info, input, objectType: input.objectType, req });
        return { __typename: "CopyResult" as const, [lowercaseFirstLetter(input.objectType)]: result };
    },
    deleteOne: async (_, { input }, { req }) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return deleteOneHelper({ input, req });
    },
    deleteMany: async (_, { input }, { req }) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return deleteManyHelper({ input, req });
    },
    deleteAll: async (_, { input }, { req }) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 25, req });
        let totalCount = 0;
        if (input.objectTypes.includes(DeleteType.RunProject)) {
            const { danger } = ModelMap.getLogic(["danger"], DeleteType.RunProject, true, "deleteAll RunProject");
            totalCount += await danger.deleteAll({ __typename: "User", id: userData.id });
        }
        if (input.objectTypes.includes(DeleteType.RunRoutine)) {
            const { danger } = ModelMap.getLogic(["danger"], DeleteType.RunRoutine, true, "deleteAll RunRoutine");
            totalCount += await danger.deleteAll({ __typename: "User", id: userData.id });
        }
        // TODO add condition for every object type. For users, make sure you can only delete bots (and not your own account)
        return {
            __typename: "Count" as const,
            count: totalCount,
        };
    },
    deleteAccount: async (_, { input }, { req, res }, info) => {
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
        const result = await deleteOneHelper({ input: { id, objectType: DeleteType.User }, req });
        // If successful, remove user from session
        if (result.success) {
            // TODO this will only work if you're deleing yourself, since it clears your own session. 
            // If there are cases like deleting a bot, or an admin deleting a user, this will need to be handled differently
            return auth.logOut(undefined, undefined as unknown as IWrap<Record<string, never>>, { req, res }, info) as any;
        } else {
            throw new CustomError("0123", "InternalError");
        }
    },
};
