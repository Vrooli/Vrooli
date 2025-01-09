import { DEFAULT_LANGUAGE, ModelType, SessionUser, TransferObjectType, TransferRequestReceiveInput, TransferRequestSendInput, TransferSortBy, transferValidation } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { permissionsSelectHelper } from "../../builders/permissionsSelectHelper";
import { PartialApiInfo, PrismaDelegate } from "../../builders/types";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { Notify } from "../../notify";
import { getSingleTypePermissions, isOwnerAdminCheck } from "../../validators";
import { TransferFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { TeamModelLogic, TransferModelInfo, TransferModelLogic, UserModelLogic } from "./types";

const __typename = "Transfer" as const;

/**
 * Maps a transferable object type to its field name in the database
 */
export const TransferableFieldMap: { [x in TransferObjectType]: string } = {
    Api: "api",
    Code: "code",
    Note: "note",
    Project: "project",
    Routine: "routine",
    Standard: "standard",
};

/**
 * Handles transferring an object from one user to another. It works like this:
 * 1. The user who owns the object creates a transfer request
 * 2. If the user is transferring to a team where they have the correct permissions, 
 *   the transfer is automatically accepted and the process is complete.
 * 3. Otherwise, a notification is sent to the user/org that is receiving the object,
 *   and they can accept or reject the transfer.
 */
export function transfer() {
    return {
        /**
         * Checks if objects being created/updated require a transfer request. Used by mutate functions 
         * of other models, so model-specific permissions checking is not required.
         * @param owners List of owners of the objects being created/updated
         * @param userData Session data of the user making the request
         * @returns List of booleans indicating if the object requires a transfer request. 
         * List is in same order as owners list.
         */
        checkTransferRequests: async (
            owners: { id: string, __typename: "Team" | "User" }[],
            userData: SessionUser,
        ): Promise<boolean[]> => {
            // Grab all create team IDs
            const orgIds = owners.filter(o => o.__typename === "Team").map(o => o.id);
            // Check if user is an admin of each team
            const isAdmins: boolean[] = await ModelMap.get<TeamModelLogic>("Team").query.hasRole(userData.id, orgIds);
            // Create return list
            const requiresTransferRequest: boolean[] = owners.map((o, i) => {
                // If owner is a user, transfer is required if user is not the same as the session user
                if (o.__typename === "User") return o.id !== userData.id;
                // If owner is a team, transfer is required if user is not an admin
                const orgIdIndex = orgIds.indexOf(o.id);
                return !isAdmins[orgIdIndex];
            });
            return requiresTransferRequest;
        },
        /**
         * Initiates a transfer request from an object you own, to another user/org
         * @returns The ID of the transfer request
         */
        requestSend: async (
            input: TransferRequestSendInput,
            userData: SessionUser,
        ): Promise<string> => {
            // Find the object and its owner
            const object: { __typename: `${TransferObjectType}`, id: string } = { __typename: input.objectType, id: input.objectConnect };
            const { dbTable, validate } = ModelMap.getLogic(["dbTable", "validate"], object.__typename);
            const permissionData = await (prismaInstance[dbTable] as PrismaDelegate).findUnique({
                where: { id: object.id },
                select: permissionsSelectHelper(validate().permissionsSelect, userData.id, userData.languages),
            });
            const owner = permissionData && validate().owner(permissionData, userData.id);
            // Check if user is allowed to transfer this object
            if (!owner || !isOwnerAdminCheck(owner, userData.id))
                throw new CustomError("0286", "NotAuthorizedToTransfer");
            // Get 'to' data
            const toType = input.toTeamConnect ? "Team" : "User";
            const toId: string = input.toTeamConnect || input.toUserConnect as string;
            // Create transfer request
            const request = await prismaInstance.transfer.create({
                data: {
                    fromUser: owner.User ? { connect: { id: owner.User.id } } : undefined,
                    fromTeam: owner.Team ? { connect: { id: owner.Team.id } } : undefined,
                    [TransferableFieldMap[object.__typename]]: { connect: { id: object.id } },
                    toUser: toType === "User" ? { connect: { id: toId } } : undefined,
                    toTeam: toType === "Team" ? { connect: { id: toId } } : undefined,
                    status: "Pending",
                    message: input.message,
                },
                select: { id: true },
            });
            // Notify user/org that is receiving the object
            const pushNotification = Notify(userData.languages).pushTransferRequestSend(request.id, object.__typename, object.id);
            if (toType === "User") await pushNotification.toUser(toId);
            else await pushNotification.toTeam(toId);
            // Return the transfer request ID
            return request.id;
        },
        /**
         * Initiates a transfer request from an object someone else owns, to you
         */
        requestReceive: async (
            info: PartialApiInfo,
            input: TransferRequestReceiveInput,
            userData: SessionUser,
        ): Promise<string> => {
            return "";//TODO
        },
        /**
         * Cancels a transfer request that you initiated
         * @param transferId The ID of the transfer request
         * @param userData The session data of the user who is cancelling the transfer request
         */
        cancel: async (transferId: string, userData: SessionUser) => {
            // Find the transfer request
            const transfer = await prismaInstance.transfer.findUnique({
                where: { id: transferId },
                select: {
                    id: true,
                    fromTeamId: true,
                    fromUserId: true,
                    toTeamId: true,
                    toUserId: true,
                    status: true,
                },
            });
            // Make sure transfer exists, and is not already accepted or rejected
            if (!transfer)
                throw new CustomError("0293", "TransferNotFound");
            if (transfer.status === "Accepted")
                throw new CustomError("0294", "TransferAlreadyAccepted");
            if (transfer.status === "Denied")
                throw new CustomError("0295", "TransferAlreadyRejected");
            // Make sure user is the owner of the transfer request
            if (transfer.fromTeamId) {
                const { validate } = ModelMap.getLogic(["validate"], "Team");
                const permissionData = await prismaInstance.team.findUnique({
                    where: { id: transfer.fromTeamId },
                    select: permissionsSelectHelper(validate().permissionsSelect, userData.id, userData.languages),
                });
                if (!permissionData || !isOwnerAdminCheck(validate().owner(permissionData, userData.id), userData.id))
                    throw new CustomError("0300", "TransferRejectNotAuthorized");
            } else if (transfer.fromUserId !== userData.id) {
                throw new CustomError("0301", "TransferRejectNotAuthorized");
            }
            // Delete transfer request
            await prismaInstance.transfer.delete({
                where: { id: transferId },
            });
        },
        /**
         * Accepts a transfer request
         * @param transferId The ID of the transfer request
         * @param userData The session data of the user who is accepting the transfer request
         */
        accept: async (transferId: string, userData: SessionUser) => {
            // Find the transfer request
            const transfer = await prismaInstance.transfer.findUnique({
                where: { id: transferId },
            });
            // Make sure transfer exists, and is not accepted or rejected
            if (!transfer)
                throw new CustomError("0287", "TransferNotFound");
            if (transfer.status === "Accepted")
                throw new CustomError("0288", "TransferAlreadyAccepted");
            if (transfer.status === "Denied")
                throw new CustomError("0289", "TransferAlreadyRejected");
            // Make sure transfer is going to you or a team you can control
            if (transfer.toTeamId) {
                const { validate } = ModelMap.getLogic(["validate"], "Team");
                const permissionData = await prismaInstance.team.findUnique({
                    where: { id: transfer.toTeamId },
                    select: permissionsSelectHelper(validate().permissionsSelect, userData.id, userData.languages),
                });
                if (!permissionData || !isOwnerAdminCheck(validate().owner(permissionData, userData.id), userData.id))
                    throw new CustomError("0302", "TransferAcceptNotAuthorized");
            } else if (transfer.toUserId !== userData.id) {
                throw new CustomError("0303", "TransferAcceptNotAuthorized");
            }
            // Transfer object, then mark transfer request as accepted
            // Find the object type, based on which relation is not null
            const typeField = [
                "apiId",
                "codeId",
                "noteId",
                "projectId",
                "routineId",
                "standardId",
            ].find((field) => transfer[field] !== null);
            if (!typeField)
                throw new CustomError("0290", "TransferMissingData");
            const type = typeField.replace("Id", "");
            await prismaInstance.transfer.update({
                where: { id: transferId },
                data: {
                    status: "Accepted",
                    [type]: {
                        update: {
                            ownerId: transfer.toUserId,
                            teamId: transfer.toTeamId,
                        },
                    },
                },
            });
            //TODO update object's hasBeenTransferred flag
            // TODO Notify user/org that sent the transfer request
            //const pushNotification = Notify(userData.languages).pushTransferAccepted(transfer.objectTitle, transferId, type);
            // if (transfer.fromUserId) await pushNotification.toUser(transfer.fromUserId);
            // else await pushNotification.toTeam(transfer.fromTeamId as string, userData.id);
        },
        /**
         * Denies a transfer request
         * @param transferId The ID of the transfer request
         * @param userData The session data of the user who is rejecting the transfer request
         * @param reason Optional reason for rejecting the transfer request
         */
        deny: async (transferId: string, userData: SessionUser, reason?: string) => {
            // Find the transfer request
            const transfer = await prismaInstance.transfer.findUnique({
                where: { id: transferId },
            });
            // Make sure transfer exists, and is not already accepted or rejected
            if (!transfer)
                throw new CustomError("0320", "TransferNotFound");
            if (transfer.status === "Accepted")
                throw new CustomError("0291", "TransferAlreadyAccepted");
            if (transfer.status === "Denied")
                throw new CustomError("0292", "TransferAlreadyRejected");
            // Make sure transfer is going to you or a team you can control
            if (transfer.toTeamId) {
                const { validate } = ModelMap.getLogic(["validate"], "Team");
                const permissionData = await prismaInstance.team.findUnique({
                    where: { id: transfer.toTeamId },
                    select: permissionsSelectHelper(validate().permissionsSelect, userData.id, userData.languages),
                });
                if (!permissionData || !isOwnerAdminCheck(validate().owner(permissionData, userData.id), userData.id))
                    throw new CustomError("0312", "TransferRejectNotAuthorized");
            } else if (transfer.toUserId !== userData.id) {
                throw new CustomError("0313", "TransferRejectNotAuthorized");
            }
            // Find the object type, based on which relation is not null
            const typeField = [
                "apiId",
                "codeId",
                "noteId",
                "projectId",
                "routineId",
                "standardId",
            ].find((field) => transfer[field] !== null);
            if (!typeField)
                throw new CustomError("0290", "TransferMissingData");
            const type = typeField.replace("Id", "");
            // Deny the transfer
            await prismaInstance.transfer.update({
                where: { id: transferId },
                data: {
                    status: "Denied",
                    denyReason: reason,
                },
            });
            // Notify user/org that sent the transfer request
            //const pushNotification = Notify(userData.languages).pushTransferRejected(transfer.objectTitle, type, transferId);
            // if (transfer.fromUserId) await pushNotification.toUser(transfer.fromUserId);
            // else await pushNotification.toTeam(transfer.fromTeamId as string, userData.id);
        },
    };
}

export const TransferModel: TransferModelLogic = ({
    __typename,
    dbTable: "transfer",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                ...Object.fromEntries(Object.entries(TransferableFieldMap).map(([key, value]) =>
                    [value, { select: ModelMap.get(key as ModelType).display().label.select() }])),
            }),
            get: (select, languages) => {
                for (const [key, value] of Object.entries(TransferableFieldMap)) {
                    if (select[value]) return ModelMap.get(key as ModelType).display().label.get(select[value], languages);
                }
                return i18next.t("common:Transfer", { lng: languages && languages.length ? languages[0] : DEFAULT_LANGUAGE });
            },
        },
    }),
    format: TransferFormat,
    mutate: {
        shape: {
            update: async ({ data }) => ({
                message: noNull(data.message),
            }),
        },
        yup: transferValidation,
    },
    search: {
        defaultSort: TransferSortBy.DateCreatedDesc,
        sortBy: TransferSortBy,
        searchFields: {
            apiId: true,
            codeId: true,
            createdTimeFrame: true,
            fromTeamId: true,
            noteId: true,
            projectId: true,
            routineId: true,
            standardId: true,
            status: true,
            toTeamId: true,
            toUserId: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                { fromUser: ModelMap.get<UserModelLogic>("User").search.searchStringQuery() },
                { fromTeam: ModelMap.get<TeamModelLogic>("Team").search.searchStringQuery() },
                { toUser: ModelMap.get<UserModelLogic>("User").search.searchStringQuery() },
                { toTeam: ModelMap.get<TeamModelLogic>("Team").search.searchStringQuery() },
                ...Object.entries(TransferableFieldMap).map(([key, value]) => ({ [value]: ModelMap.getLogic(["search"], key as ModelType).search.searchStringQuery() })),
            ],
        }),
        supplemental: {
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<TransferModelInfo["ApiPermission"]>(__typename, ids, userData)),
                    },
                };
            },
        },
    },
    transfer,
    validate: () => ({}) as any,
});
