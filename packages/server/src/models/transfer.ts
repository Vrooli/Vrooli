import { isObject } from "@shared/utils";
import { CustomError } from "../events";
import { resolveUnion } from "../endpoints/resolvers";
import { SessionUser, Transfer, TransferObjectType, TransferRequestReceiveInput, TransferRequestSendInput, TransferSearchInput, TransferSortBy, TransferUpdateInput, Vote } from "../endpoints/types";
import { PrismaType } from "../types";
import { readManyHelper } from "../actions";
import { Displayer, Formatter, GraphQLModelType, Mutater } from "./types";
import { ApiModel, CommentModel, NoteModel, ProjectModel, RoutineModel, SmartContractModel, StandardModel } from ".";
import { PartialGraphQLInfo, SelectWrap } from "../builders/types";
import { padSelect } from "../builders";
import { Prisma } from "@prisma/client";
import { GraphQLResolveInfo } from "graphql";
import { getDelegator, getValidator } from "../getters";
import { isOwnerAdminCheck } from "../validators";
import { Notify } from "../notify";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlModel: Transfer,
    GqlSearch: TransferSearchInput,
    GqlSort: TransferSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.transferUpsertArgs['create'],
    PrismaUpdate: Prisma.transferUpsertArgs['update'],
    PrismaModel: Prisma.transferGetPayload<SelectWrap<Prisma.transferSelect>>,
    PrismaSelect: Prisma.transferSelect,
    PrismaWhere: Prisma.transferWhereInput,
}

const __typename = 'Transfer' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        fromOwner: {
            fromUser: 'User',
            fromOrganization: 'Organization',
        },
        toOwner: {
            toUser: 'User',
            toOrganization: 'Organization',
        },
        object: {
            api: 'Api',
            note: 'Note',
            project: 'Project',
            routine: 'Routine',
            smartContract: 'SmartContract',
            standard: 'Standard',
        }
    },
})

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
const transfer = (prisma: PrismaType) => ({
    /**
     * Initiates a transfer request from an object you own, to another user/org
     * @returns The ID of the transfer request
     */
    requestSend: async (
        info: GraphQLResolveInfo | PartialGraphQLInfo,
        input: TransferRequestSendInput,
        userData: SessionUser,
    ): Promise<string> => {
        // Find the object and its owner
        const object: { __typename: TransferObjectType, id: string } = { __typename: input.objectType, id: input.objectId };
        const validator = getValidator(object.__typename, userData.languages, 'Transfer.request-object');
        const prismaDelegate = getDelegator(object.__typename, prisma, userData.languages, 'Transfer.request-object');
        const permissionData = await prismaDelegate.findUnique({
            where: { id: object.id },
            select: validator.permissionsSelect,
        });
        const owner = permissionData && validator.owner(permissionData);
        // Check if user is allowed to transfer this object
        if (!owner || !isOwnerAdminCheck(owner, userData.id))
            throw new CustomError('0286', 'NotAuthorizedToTransfer', userData.languages);
        // Check if the user is transferring to themselves
        const toType = input.toOrganizationId ? 'Organization' : 'User';
        const toId: string = input.toOrganizationId || input.toUserId as string;
        const toValidator = getValidator(toType, userData.languages, 'Transfer.request-validator');
        const toPrismaDelegate = getDelegator(toType, prisma, userData.languages, 'Transfer.request-validator');
        const toPermissionData = await toPrismaDelegate.findUnique({
            where: { id: toId },
            select: toValidator.permissionsSelect(userData.id, userData.languages),
        });
        const isAdmin = toPermissionData && isOwnerAdminCheck(toValidator.owner(toPermissionData), userData.id)
        // Create transfer request
        const request = await prisma.transfer.create({
            data: {
                fromUser: owner.User ? { connect: { id: owner.User.id } } : undefined,
                fromOrganization: owner.Organization ? { connect: { id: owner.Organization.id } } : undefined,
                [TransferableFieldMap[object.__typename]]: { connect: { id: object.id } },
                toUser: toType === 'User' ? { connect: { id: toId } } : undefined,
                toOrganization: toType === 'Organization' ? { connect: { id: toId } } : undefined,
                status: isAdmin ? 'Accepted' : 'Pending',
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
            const validator = getValidator('Organization', userData.languages, 'Transfer.cancel');
            const permissionData = await prisma.organization.findUnique({
                where: { id: transfer.fromOrganizationId },
                select: validator.permissionsSelect(userData.id, userData.languages),
            });
            if (!permissionData || !isOwnerAdminCheck(validator.owner(permissionData), userData.id))
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
            const validator = getValidator('Organization', userData.languages, 'Transfer.accept');
            const permissionData = await prisma.organization.findUnique({
                where: { id: transfer.toOrganizationId },
                select: validator.permissionsSelect(userData.id, userData.languages),
            });
            if (!permissionData || !isOwnerAdminCheck(validator.owner(permissionData), userData.id))
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
        // Notify user/org that sent the transfer request
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
            const validator = getValidator('Organization', userData.languages, 'Transfer.reject');
            const permissionData = await prisma.organization.findUnique({
                where: { id: transfer.toOrganizationId },
                select: validator.permissionsSelect(userData.id, userData.languages),
            });
            if (!permissionData || !isOwnerAdminCheck(validator.owner(permissionData), userData.id))
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

const mutater = (): Mutater<Model> => ({
    shape: {
        update: async ({ data }) => ({
            id: data.id,
            message: data.message
        }),
    },
    yup: { update: {} as any },
})

const displayer = (): Displayer<Model> => ({
    select: () => ({
        id: true,
        api: padSelect(ApiModel.display.select),
        note: padSelect(NoteModel.display.select),
        project: padSelect(ProjectModel.display.select),
        routine: padSelect(RoutineModel.display.select),
        smartContract: padSelect(SmartContractModel.display.select),
        standard: padSelect(StandardModel.display.select),
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
})

export const TransferModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.transfer,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    transfer,
    validate: {} as any,
})