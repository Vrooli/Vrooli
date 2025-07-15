// AI_CHECK: TASK_ID=type-safety-fixes | LAST: 2025-06-29
import { DeleteType, lowercaseFirstLetter } from "@vrooli/shared";
import type { CopyInput, CopyResult, Count, DeleteAccountInput, DeleteAllInput, DeleteManyInput, DeleteOneInput, Success } from "@vrooli/shared";
import { copyHelper } from "../../actions/copies.js";
import { deleteManyHelper, deleteOneHelper } from "../../actions/deletes.js";
import { PasswordAuthService } from "../../auth/email.js";
import { RequestService } from "../../auth/request.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { ModelMap } from "../../models/base/index.js";
import { type ApiEndpoint } from "../../types.js";
import { auth } from "./auth.js";

export type EndpointsActions = {
    copy: ApiEndpoint<CopyInput, CopyResult>;
    deleteOne: ApiEndpoint<DeleteOneInput, Success>;
    deleteMany: ApiEndpoint<DeleteManyInput, Count>;
    deleteAll: ApiEndpoint<DeleteAllInput, Count>;
    deleteAccount: ApiEndpoint<DeleteAccountInput, Success>;
}

export const actions: EndpointsActions = {
    copy: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        const result = await copyHelper({ info, input, objectType: input.objectType, req });
        return { __typename: "CopyResult" as const, [lowercaseFirstLetter(input.objectType)]: result };
    },
    deleteOne: async ({ input }, { req }) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        return deleteOneHelper({ input, req });
    },
    deleteMany: async ({ input }, { req }) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        return deleteManyHelper({ input, req });
    },
    deleteAll: async ({ input }, { req }) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 25, req });
        RequestService.assertRequestFrom(req, { hasWriteAuthPermissions: true });
        let totalCount = 0;
        if (input.objectTypes.includes(DeleteType.Run)) {
            // AI_CHECK: TYPE_SAFETY=phase1-2 | LAST: 2025-07-03 - Replaced deprecated RunProject/RunRoutine with unified Run type
            const { danger } = ModelMap.getLogic(["danger"], DeleteType.Run, true, "deleteAll Run");
            totalCount += await danger.deleteAll({ __typename: "User", id: userData.id });
        }
        // TODO add condition for every object type. For users, make sure you can only delete bots (and not your own account)
        return {
            __typename: "Count" as const,
            count: totalCount,
        };
    },
    deleteAccount: async ({ input }, { req, res }, _info) => {
        const { id } = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 500, req });
        RequestService.assertRequestFrom(req, { hasWriteAuthPermissions: true });
        // Find user
        const user = await DbProvider.get().user.findUnique({
            where: { id: BigInt(id) },
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
            // Log out the user session
            await auth.logOut({} as never, { req, res }, {});
            // Return Success instead of Session
            return {
                __typename: "Success" as const,
                success: true,
            };
        } else {
            throw new CustomError("0123", "InternalError");
        }
    },
};

// AI_CHECK: TYPE_SAFETY=api-endpoints-1 | LAST: 2025-07-03 - Fixed bigint ID conversions, user object type casting, and deleteAccount return type from Session to Success
