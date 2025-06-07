import { type FindByIdInput, type MeetingInvite, type MeetingInviteCreateInput, type MeetingInviteSearchInput, type MeetingInviteSearchResult, type MeetingInviteUpdateInput, validatePK } from "@vrooli/shared";
import { createManyHelper, createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateManyHelper, updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { SessionService } from "../../auth/session.js";
import { InfoConverter } from "../../builders/infoConverter.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { ModelMap } from "../../models/base/index.js";
import { type MeetingInviteModelLogic } from "../../models/base/types.js";
import { type ApiEndpoint } from "../../types.js";

export type EndpointsMeetingInvite = {
    findOne: ApiEndpoint<FindByIdInput, MeetingInvite>;
    findMany: ApiEndpoint<MeetingInviteSearchInput, MeetingInviteSearchResult>;
    createOne: ApiEndpoint<MeetingInviteCreateInput, MeetingInvite>;
    createMany: ApiEndpoint<MeetingInviteCreateInput[], MeetingInvite[]>;
    updateOne: ApiEndpoint<MeetingInviteUpdateInput, MeetingInvite>;
    updateMany: ApiEndpoint<MeetingInviteUpdateInput[], MeetingInvite[]>;
    acceptOne: ApiEndpoint<FindByIdInput, MeetingInvite>;
    declineOne: ApiEndpoint<FindByIdInput, MeetingInvite>;
}

const objectType = "MeetingInvite";
export const meetingInvite: EndpointsMeetingInvite = {
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
        await RequestService.get().rateLimit({ maxUser: 100, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        if (!input || !input.id || !validatePK(input.id)) {
            throw new CustomError("0400", "InvalidArgs");
        }
        const userData = SessionService.getUser(req);
        if (!userData) throw new CustomError("0401", "Unauthorized");

        const model = ModelMap.get<MeetingInviteModelLogic>(objectType);
        const partialInfo = InfoConverter.get().fromApiToPartialApi(info, model.format.apiRelMap, true);

        const invite = await DbProvider.get().meeting_invite.findUnique({
            where: { id: BigInt(input.id) },
            select: { id: true, userId: true, meetingId: true, status: true },
        });

        if (!invite) throw new CustomError("0404", "NotFound", { objectType });
        if (invite.userId.toString() !== userData.id) throw new CustomError("0803", "Unauthorized");
        if (invite.status !== "Pending") throw new CustomError("0809", "InternalError");

        await DbProvider.get().$transaction([
            DbProvider.get().meeting_invite.update({
                where: { id: invite.id },
                data: { status: "Accepted" },
            }),
            DbProvider.get().meeting_attendees.upsert({
                where: { meeting_attendees_meetingid_userid_unique: { meetingId: invite.meetingId, userId: invite.userId } },
                create: { meetingId: invite.meetingId, userId: invite.userId },
                update: {},
                select: { id: true },
            }),
        ]);

        const result = await readOneHelper({ info: partialInfo, input: { id: invite.id.toString() }, objectType, req });
        return result as MeetingInvite;
    },
    declineOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        if (!input || !input.id || !validatePK(input.id)) {
            throw new CustomError("0400", "InvalidArgs");
        }
        const userData = SessionService.getUser(req);
        if (!userData) throw new CustomError("0401", "Unauthorized");

        const model = ModelMap.get<MeetingInviteModelLogic>(objectType);
        const partialInfo = InfoConverter.get().fromApiToPartialApi(info, model.format.apiRelMap, true);

        const invite = await DbProvider.get().meeting_invite.findUnique({
            where: { id: BigInt(input.id) },
            select: { id: true, userId: true, meetingId: true, status: true },
        });

        if (!invite) throw new CustomError("0404", "NotFound", { objectType });
        if (invite.userId.toString() !== userData.id) throw new CustomError("0803", "Unauthorized");
        if (invite.status !== "Pending") throw new CustomError("0809", "InternalError");

        await DbProvider.get().meeting_invite.update({
            where: { id: invite.id },
            data: { status: "Declined" },
        });

        const result = await readOneHelper({ info: partialInfo, input: { id: invite.id.toString() }, objectType, req });
        return result as MeetingInvite;
    },
};
