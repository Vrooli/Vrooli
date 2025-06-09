import { type FindByIdInput, generatePublicId, type MemberInvite, type MemberInviteCreateInput, type MemberInviteSearchInput, type MemberInviteSearchResult, type MemberInviteUpdateInput, validatePK } from "@vrooli/shared";
import { readOneHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { SessionService } from "../../auth/session.js";
import { InfoConverter } from "../../builders/infoConverter.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { ModelMap } from "../../models/base/index.js";
import { type MemberInviteModelLogic } from "../../models/base/types.js";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsMemberInvite = {
    findOne: ApiEndpoint<FindByIdInput, MemberInvite>;
    findMany: ApiEndpoint<MemberInviteSearchInput, MemberInviteSearchResult>;
    createOne: ApiEndpoint<MemberInviteCreateInput, MemberInvite>;
    createMany: ApiEndpoint<MemberInviteCreateInput[], MemberInvite[]>;
    updateOne: ApiEndpoint<MemberInviteUpdateInput, MemberInvite>;
    updateMany: ApiEndpoint<MemberInviteUpdateInput[], MemberInvite[]>;
    acceptOne: ApiEndpoint<FindByIdInput, MemberInvite>;
    declineOne: ApiEndpoint<FindByIdInput, MemberInvite>;
}

const objectType = "MemberInvite";
export const memberInvite: EndpointsMemberInvite = createStandardCrudEndpoints({
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

            const model = ModelMap.get<MemberInviteModelLogic>(objectType);
            const partialInfo = InfoConverter.get().fromApiToPartialApi(info, model.format.apiRelMap, true);

            const invite = await DbProvider.get().member_invite.findUnique({
                where: { id: BigInt(input.id) },
                select: { id: true, userId: true, teamId: true, status: true },
            });

            if (!invite) throw new CustomError("0404", "NotFound", { objectType });
            if (invite.userId.toString() !== userData.id) throw new CustomError("0803", "Unauthorized");
            if (invite.status !== "Pending") throw new CustomError("0809", "InternalError");

            await DbProvider.get().$transaction([
                DbProvider.get().member_invite.update({
                    where: { id: invite.id },
                    data: { status: "Accepted" },
                }),
                DbProvider.get().member.upsert({
                    where: {
                        member_teamid_userid_unique: {
                            teamId: invite.teamId,
                            userId: invite.userId,
                        },
                    },
                    create: {
                        publicId: generatePublicId(),
                        teamId: invite.teamId,
                        userId: invite.userId,
                    },
                    update: {},
                    select: { id: true },
                }),
            ]);

            const result = await readOneHelper({ info: partialInfo, input: { id: invite.id.toString() }, objectType, req });
            return result as MemberInvite;
        },
        declineOne: async ({ input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
            if (!input || !input.id || !validatePK(input.id)) {
                throw new CustomError("0400", "InvalidArgs");
            }
            const userData = SessionService.getUser(req);
            if (!userData) throw new CustomError("0401", "Unauthorized");

            const model = ModelMap.get<MemberInviteModelLogic>(objectType);
            const partialInfo = InfoConverter.get().fromApiToPartialApi(info, model.format.apiRelMap, true);

            const invite = await DbProvider.get().member_invite.findUnique({
                where: { id: BigInt(input.id) },
                select: { id: true, userId: true, teamId: true, status: true },
            });

            if (!invite) throw new CustomError("0404", "NotFound", { objectType });
            if (invite.userId.toString() !== userData.id) throw new CustomError("0803", "Unauthorized");
            if (invite.status !== "Pending") throw new CustomError("0809", "InternalError");

            await DbProvider.get().member_invite.update({
                where: { id: invite.id },
                data: { status: "Declined" },
            });

            const result = await readOneHelper({ info: partialInfo, input: { id: invite.id.toString() }, objectType, req });
            return result as MemberInvite;
        },
    },
});