import { type ChatInvite, type ChatInviteCreateInput, type ChatInviteSearchInput, type ChatInviteSearchResult, type ChatInviteUpdateInput, type FindByIdInput, validatePK } from "@local/shared";
import { createManyHelper, createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateManyHelper, updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { SessionService } from "../../auth/session.js";
import { InfoConverter } from "../../builders/infoConverter.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { ModelMap } from "../../models/base/index.js";
import { type ChatInviteModelLogic } from "../../models/base/types.js";
import { type ApiEndpoint } from "../../types.js";

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
export const chatInvite: EndpointsChatInvite = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
        return readManyHelper({ info, input, objectType, req });
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        return createOneHelper({ info, input, objectType, req });
    },
    createMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        return createManyHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        return updateOneHelper({ info, input, objectType, req });
    },
    updateMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        return updateManyHelper({ info, input, objectType, req });
    },
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
};
