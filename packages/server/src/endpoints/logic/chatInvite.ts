import { type ChatInvite, type ChatInviteCreateInput, type ChatInviteSearchInput, type ChatInviteSearchResult, type ChatInviteUpdateInput, type FindByIdInput, validatePK } from "@vrooli/shared";
import { readOneHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { SessionService } from "../../auth/session.js";
import { InfoConverter } from "../../builders/infoConverter.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { ModelMap } from "../../models/base/index.js";
import { type ChatInviteModelLogic } from "../../models/base/types.js";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsChatInvite = {
    findOne: ApiEndpoint<FindByIdInput, ChatInvite>;
    findMany: ApiEndpoint<ChatInviteSearchInput, ChatInviteSearchResult>;
    createOne: ApiEndpoint<ChatInviteCreateInput, ChatInvite>;
    createMany: ApiEndpoint<ChatInviteCreateInput[], ChatInvite[]>;
    updateOne: ApiEndpoint<ChatInviteUpdateInput, ChatInvite>;
    updateMany: ApiEndpoint<ChatInviteUpdateInput[], ChatInvite[]>;
    acceptOne: ApiEndpoint<FindByIdInput, ChatInvite>;
    declineOne: ApiEndpoint<FindByIdInput, ChatInvite>;
}

const objectType = "ChatInvite";
export const chatInvite: EndpointsChatInvite = createStandardCrudEndpoints({
    objectType,
    endpoints: {
        findOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
        createOne: {
            rateLimit: RateLimitPresets.STRICT,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
        createMany: {
            rateLimit: RateLimitPresets.STRICT,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
        updateOne: {
            rateLimit: RateLimitPresets.LOW,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
        updateMany: {
            rateLimit: RateLimitPresets.LOW,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
    },
    customEndpoints: {
        acceptOne: async ({ input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
            if (!input || !input.id || !validatePK(input.id)) {
                throw new CustomError("0400", "InvalidArgs");
            }
            const userData = SessionService.getUser(req);
            if (!userData) throw new CustomError("0401", "Unauthorized");

            const model = ModelMap.get<ChatInviteModelLogic>(objectType);
            const partialInfo = InfoConverter.get().fromApiToPartialApi(info, model.format.apiRelMap, true);

            const invite = await DbProvider.get().chat_invite.findUnique({
                where: { id: BigInt(input.id) },
                select: { id: true, userId: true, chatId: true, status: true },
            });

            if (!invite) throw new CustomError("0404", "NotFound", { objectType });
            if (invite.userId.toString() !== userData.id) throw new CustomError("0803", "Unauthorized");
            if (invite.status !== "Pending") throw new CustomError("0809", "InternalError");

            await DbProvider.get().$transaction([
                DbProvider.get().chat_invite.update({
                    where: { id: invite.id },
                    data: { status: "Accepted" },
                }),
                DbProvider.get().chat_participants.upsert({
                    where: {
                        chat_participants_chatid_userid_unique: {
                            chatId: invite.chatId,
                            userId: invite.userId,
                        },
                    },
                    create: {
                        chatId: invite.chatId,
                        userId: invite.userId,
                    },
                    update: {},
                    select: { id: true },
                }),
            ]);

            const result = await readOneHelper({ info: partialInfo, input: { id: invite.id.toString() }, objectType, req });
            return result as ChatInvite;
        },
        declineOne: async ({ input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
            if (!input || !input.id || !validatePK(input.id)) {
                throw new CustomError("0400", "InvalidArgs");
            }
            const userData = SessionService.getUser(req);
            if (!userData) throw new CustomError("0401", "Unauthorized");

            const model = ModelMap.get<ChatInviteModelLogic>(objectType);
            const partialInfo = InfoConverter.get().fromApiToPartialApi(info, model.format.apiRelMap, true);

            const invite = await DbProvider.get().chat_invite.findUnique({
                where: { id: BigInt(input.id) },
                select: { id: true, userId: true, chatId: true, status: true },
            });

            if (!invite) throw new CustomError("0404", "NotFound", { objectType });
            if (invite.userId.toString() !== userData.id) throw new CustomError("0803", "Unauthorized");
            if (invite.status !== "Pending") throw new CustomError("0809", "InternalError");

            await DbProvider.get().chat_invite.update({
                where: { id: invite.id },
                data: { status: "Declined" },
            });

            const result = await readOneHelper({ info: partialInfo, input: { id: invite.id.toString() }, objectType, req });
            return result as ChatInvite;
        },
    },
});
