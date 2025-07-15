import { type FindByIdInput, type MeetingInvite, type MeetingInviteCreateInput, type MeetingInviteSearchInput, type MeetingInviteSearchResult, type MeetingInviteUpdateInput, validatePK } from "@vrooli/shared";
import { readOneHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { SessionService } from "../../auth/session.js";
import { InfoConverter } from "../../builders/infoConverter.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { ModelMap } from "../../models/base/index.js";
import { type MeetingInviteModelLogic } from "../../models/base/types.js";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

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
export const meetingInvite: EndpointsMeetingInvite = createStandardCrudEndpoints({
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
        acceptOne: async (data, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
            const input = data?.input;
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
                    create: { id: invite.id, meetingId: invite.meetingId, userId: invite.userId },
                    update: {},
                    select: { id: true },
                }),
            ]);

            const result = await readOneHelper({ info: partialInfo, input: { id: invite.id.toString() }, objectType, req });
            return result as MeetingInvite;
        },
        declineOne: async (data, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
            const input = data?.input;
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
    },
});
