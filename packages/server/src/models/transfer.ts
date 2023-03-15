import { CustomError } from "../events";
import { SessionUser, Transfer, TransferObjectType, TransferRequestReceiveInput, TransferRequestSendInput, TransferSearchInput, TransferSortBy, TransferUpdateInput, TransferYou } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { ApiModel, NoteModel, OrganizationModel, ProjectModel, RoutineModel, SmartContractModel, StandardModel } from ".";
import { PartialGraphQLInfo, SelectWrap } from "../builders/types";
import { selPad, permissionsSelectHelper, noNull } from "../builders";
import { Prisma } from "@prisma/client";
import { GraphQLResolveInfo } from "graphql";
import { getLogic } from "../getters";
import { getSingleTypePermissions, isOwnerAdminCheck } from "../validators";
import { Notify } from "../notify";
import { transferValidation } from "@shared/validation";

const __typename = 'Transfer' as const;
type Permissions = Pick<TransferYou, 'canDelete' | 'canUpdate'>;
const suppFields = [] as const;

/**
 * Maps a transferable object type to its field name in the database
 */
export const TransferableFieldMap: { [x in TransferObjectType]: string } = {
    Api: 'api',
    Note: 'note',
    Project: 'project',
    Routine: 'routine',
    SmartContract: 'smartContract',
    Standard: 'standard',
};

/**
 * Handles transferring an object from one user to another. It works like this:
 * 1. The user who owns the object creates a transfer request
 * 2. If the user is transferring to an organization where they have the correct permissions, 
 *   the transfer is automatically accepted and the process is complete.
 * 3. Otherwise, a notification is sent to the user/org that is receiving the object,
 *   and they can accept or reject the transfer.
 */
export const transfer = (prisma: PrismaType) => ({
    /**
     * Checks if objects being created/updated require a transfer request. Used by mutate functions 
     * of other models, so model-specific permissions checking is not required.
     * @param owners List of owners of the objects being created/updated
     * @param userData Session data of the user making the request
     * @returns List of booleans indicating if the object requires a transfer request. 
     * List is in same order as owners list.
     */
    checkTransferRequests: async (
        owners: { id: string, __typename: 'Organization' | 'User' }[],
        userData: SessionUser,
    ): Promise<boolean[]> => {
        // Grab all create organization IDs
        const orgIds = owners.filter(o => o.__typename === 'Organization').map(o => o.id);
        // Check if user is an admin of each organization
        const isAdmins: boolean[] = await OrganizationModel.query.hasRole(prisma, userData.id, orgIds);
        // Create return list
        console.log('checking transfer requests', orgIds, userData.id, JSON.stringify(owners), '\n\n');
        const requiresTransferRequest: boolean[] = owners.map((o, i) => {
            // If owner is a user, transfer is required if user is not the same as the session user
            if (o.__typename === 'User') return o.id !== userData.id;
            // If owner is an organization, transfer is required if user is not an admin
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
        const { delegate, validate } = getLogic(['delegate', 'validate'], object.__typename, userData.languages, 'Transfer.request-object');
        const permissionData = await delegate(prisma).findUnique({
            where: { id: object.id },
            select: validate.permissionsSelect,
        });
        const owner = permissionData && validate.owner(permissionData);
        // Check if user is allowed to transfer this object
        if (!owner || !isOwnerAdminCheck(owner, userData.id))
            throw new CustomError('0286', 'NotAuthorizedToTransfer', userData.languages);
        // Get 'to' data
        const toType = input.toOrganizationConnect ? 'Organization' : 'User';
        const toId: string = input.toOrganizationConnect || input.toUserConnect as string;
        // Create transfer request
        const request = await prisma.transfer.create({
            data: {
                fromUser: owner.User ? { connect: { id: owner.User.id } } : undefined,
                fromOrganization: owner.Organization ? { connect: { id: owner.Organization.id } } : undefined,
                [TransferableFieldMap[object.__typename]]: { connect: { id: object.id } },
                toUser: toType === 'User' ? { connect: { id: toId } } : undefined,
                toOrganization: toType === 'Organization' ? { connect: { id: toId } } : undefined,
                status: 'Pending',
                message: input.message,
            },
            select: { id: true }
        });
        // Notify user/org that is receiving the object
        const pushNotification = Notify(prisma, userData.languages).pushTransferRequestSend(request.id, object.__typename, object.id);
        if (toType === 'User') await pushNotification.toUser(toId);
        else await pushNotification.toOrganization(toId);
        // Return the transfer request ID
        return request.id;
    },
    /**
     * Initiates a transfer request from an object someone else owns, to you
     */
    requestReceive: async (
        info: GraphQLResolveInfo | PartialGraphQLInfo,
        input: TransferRequestReceiveInput,
        userData: SessionUser,
    ): Promise<string> => {
        return '';//TODO
    },
    /**
     * Cancels a transfer request that you initiated
     * @param transferId The ID of the transfer request
     * @param userData The session data of the user who is cancelling the transfer request
     */
    cancel: async (transferId: string, userData: SessionUser) => {
        // Find the transfer request
        const transfer = await prisma.transfer.findUnique({
            where: { id: transferId },
            select: {
                id: true,
                fromOrganizationId: true,
                fromUserId: true,
                toOrganizationId: true,
                toUserId: true,
                status: true,
            }
        });
        // Make sure transfer exists, and is not already accepted or rejected
        if (!transfer)
            throw new CustomError('0293', 'TransferNotFound', userData.languages);
        if (transfer.status === 'Accepted')
            throw new CustomError('0294', 'TransferAlreadyAccepted', userData.languages);
        if (transfer.status === 'Denied')
            throw new CustomError('0295', 'TransferAlreadyRejected', userData.languages);
        // Make sure user is the owner of the transfer request
        if (transfer.fromOrganizationId) {
            const { validate } = getLogic(['validate'], 'Organization', userData.languages, 'Transfer.cancel');
            const permissionData = await prisma.organization.findUnique({
                where: { id: transfer.fromOrganizationId },
                select: permissionsSelectHelper(validate.permissionsSelect, userData.id, userData.languages),
            });
            if (!permissionData || !isOwnerAdminCheck(validate.owner(permissionData), userData.id))
                throw new CustomError('0300', 'TransferRejectNotAuthorized', userData.languages);
        } else if (transfer.fromUserId !== userData.id) {
            throw new CustomError('0301', 'TransferRejectNotAuthorized', userData.languages);
        }
        // Delete transfer request
        await prisma.transfer.delete({
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
        const transfer = await prisma.transfer.findUnique({
            where: { id: transferId },
        });
        // Make sure transfer exists, and is not accepted or rejected
        if (!transfer)
            throw new CustomError('0287', 'TransferNotFound', userData.languages);
        if (transfer.status === 'Accepted')
            throw new CustomError('0288', 'TransferAlreadyAccepted', userData.languages);
        if (transfer.status === 'Denied')
            throw new CustomError('0289', 'TransferAlreadyRejected', userData.languages);
        // Make sure transfer is going to you or an organization you can control
        if (transfer.toOrganizationId) {
            const { validate } = getLogic(['validate'], 'Organization', userData.languages, 'Transfer.accept');
            const permissionData = await prisma.organization.findUnique({
                where: { id: transfer.toOrganizationId },
                select: permissionsSelectHelper(validate.permissionsSelect, userData.id, userData.languages),
            });
            if (!permissionData || !isOwnerAdminCheck(validate.owner(permissionData), userData.id))
                throw new CustomError('0302', 'TransferAcceptNotAuthorized', userData.languages);
        } else if (transfer.toUserId !== userData.id) {
            throw new CustomError('0303', 'TransferAcceptNotAuthorized', userData.languages);
        }
        // Transfer object, then mark transfer request as accepted
        // Find the object type, based on which relation is not null
        const typeField = ['apiId', 'noteId', 'projectId', 'routineId', 'smartContractId', 'standardId'].find((field) => transfer[field] !== null);
        if (!typeField)
            throw new CustomError('0290', 'TransferMissingData', userData.languages);
        const type = typeField.replace('Id', '');
        await prisma.transfer.update({
            where: { id: transferId },
            data: {
                status: 'Accepted',
                [type]: {
                    update: {
                        ownerId: transfer.toUserId,
                        organizationId: transfer.toOrganizationId,
                    }
                }
            }
        });
        //TODO update object's hasBeenTransferred flag
        // TODO Notify user/org that sent the transfer request
        //const pushNotification = Notify(prisma, userData.languages).pushTransferAccepted(transfer.objectTitle, transferId, type);
        // if (transfer.fromUserId) await pushNotification.toUser(transfer.fromUserId);
        // else await pushNotification.toOrganization(transfer.fromOrganizationId as string, userData.id);
    },
    /**
     * Denies a transfer request
     * @param transferId The ID of the transfer request
     * @param userData The session data of the user who is rejecting the transfer request
     * @param reason Optional reason for rejecting the transfer request
     */
    deny: async (transferId: string, userData: SessionUser, reason?: string) => {
        // Find the transfer request
        const transfer = await prisma.transfer.findUnique({
            where: { id: transferId },
        });
        // Make sure transfer exists, and is not already accepted or rejected
        if (!transfer)
            throw new CustomError('0320', 'TransferNotFound', userData.languages);
        if (transfer.status === 'Accepted')
            throw new CustomError('0291', 'TransferAlreadyAccepted', userData.languages);
        if (transfer.status === 'Denied')
            throw new CustomError('0292', 'TransferAlreadyRejected', userData.languages);
        // Make sure transfer is going to you or an organization you can control
        if (transfer.toOrganizationId) {
            const { validate } = getLogic(['validate'], 'Organization', userData.languages, 'Transfer.reject');
            const permissionData = await prisma.organization.findUnique({
                where: { id: transfer.toOrganizationId },
                select: permissionsSelectHelper(validate.permissionsSelect, userData.id, userData.languages),
            });
            if (!permissionData || !isOwnerAdminCheck(validate.owner(permissionData), userData.id))
                throw new CustomError('0312', 'TransferRejectNotAuthorized', userData.languages);
        } else if (transfer.toUserId !== userData.id) {
            throw new CustomError('0313', 'TransferRejectNotAuthorized', userData.languages);
        }
        // Find the object type, based on which relation is not null
        const typeField = ['apiId', 'noteId', 'projectId', 'routineId', 'smartContractId', 'standardId'].find((field) => transfer[field] !== null);
        if (!typeField)
            throw new CustomError('0290', 'TransferMissingData', userData.languages);
        const type = typeField.replace('Id', '');
        // Deny the transfer
        await prisma.transfer.update({
            where: { id: transferId },
            data: {
                status: 'Denied',
                denyReason: reason,
            }
        });
        // Notify user/org that sent the transfer request
        //const pushNotification = Notify(prisma, userData.languages).pushTransferRejected(transfer.objectTitle, type, transferId);
        // if (transfer.fromUserId) await pushNotification.toUser(transfer.fromUserId);
        // else await pushNotification.toOrganization(transfer.fromOrganizationId as string, userData.id);
    },
})

export const TransferModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: TransferUpdateInput,
    GqlModel: Transfer,
    GqlSearch: TransferSearchInput,
    GqlSort: TransferSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.transferUpsertArgs['create'],
    PrismaUpdate: Prisma.transferUpsertArgs['update'],
    PrismaModel: Prisma.transferGetPayload<SelectWrap<Prisma.transferSelect>>,
    PrismaSelect: Prisma.transferSelect,
    PrismaWhere: Prisma.transferWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.transfer,
    display: {
        select: () => ({
            id: true,
            api: selPad(ApiModel.display.select),
            note: selPad(NoteModel.display.select),
            project: selPad(ProjectModel.display.select),
            routine: selPad(RoutineModel.display.select),
            smartContract: selPad(SmartContractModel.display.select),
            standard: selPad(StandardModel.display.select),
        }),
        label: (select, languages) => {
            if (select.api) return ApiModel.display.label(select.api as any, languages);
            if (select.note) return NoteModel.display.label(select.note as any, languages);
            if (select.project) return ProjectModel.display.label(select.project as any, languages);
            if (select.routine) return RoutineModel.display.label(select.routine as any, languages);
            if (select.smartContract) return SmartContractModel.display.label(select.smartContract as any, languages);
            if (select.standard) return StandardModel.display.label(select.standard as any, languages);
            return '';
        }
    },
    format: {
        gqlRelMap: {
            __typename,
            fromOwner: {
                fromUser: 'User',
                fromOrganization: 'Organization',
            },
            object: {
                api: 'Api',
                note: 'Note',
                project: 'Project',
                routine: 'Routine',
                smartContract: 'SmartContract',
                standard: 'Standard',
            },
            toOwner: {
                toUser: 'User',
                toOrganization: 'Organization',
            },
        },
        prismaRelMap: {
            __typename,
            fromUser: 'User',
            fromOrganization: 'Organization',
            toUser: 'User',
            toOrganization: 'Organization',
            api: 'Api',
            note: 'Note',
            project: 'Project',
            routine: 'Routine',
            smartContract: 'SmartContract',
            standard: 'Standard',
        },
        countFields: {},
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    }
                }
            },
        },
    },
    mutate: {
        shape: {
            update: async ({ data }) => ({
                message: noNull(data.message),
            })
        },
        yup: transferValidation,
    },
    transfer,
    validate: {} as any,
})