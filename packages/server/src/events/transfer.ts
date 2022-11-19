import { CODE } from "@shared/consts";
import { getDelegate, getValidator } from "../models/utils";
import { isOwnerAdminCheck } from "../models/validators/isOwnerAdminCheck";
import { Notify } from "../notify";
import { PrismaType } from "../types";
import { CustomError } from "./error";
import { genErrorCode } from "./logger";

export type TransferableObjects = 'Project' | 'Routine' | 'Standard'; //'Api' | 'Note' | 'Project' | 'Routine' | 'SmartContract' | 'Standard';

/**
 * Maps a transferable object type to its field name in the database
 */
export const TransferableFieldMap: { [x in TransferableObjects]: string } = {
    // Api: 'api',
    // Note: 'note',
    Project: 'project',
    Routine: 'routine',
    // SmartContract: 'smartContract',
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
export const Transfer = (prisma: PrismaType) => ({
    /**
     * Initiates a transfer request from one user to another user or organization
     * @param to The user/org that is receiving the object
     * @param object The object being transferred
     * @param userId The user who is transferring the object. We will check if this user is the owner of the object.
     * @returns True if transfer is complete (i.e. the user is transferring to an organization where they have the correct permissions), 
     * false if the transfer is pending
     */
    request: async (
        to: { __typename: 'Organization' | 'User', id: string },
        object: { __typename: TransferableObjects, id: string },
        userId: string,
        message?: string,
    ): Promise<boolean> => {
        // Find the object and its owner
        const validator = getValidator(object.__typename, 'Transfer.request-object');
        const prismaDelegate = getDelegate(object.__typename, prisma, 'Transfer.request-object');
        const permissionData = await prismaDelegate.findUnique({
            where: { id: object.id },
            select: validator.permissionsSelect,
        });
        const owner = permissionData && validator.owner(permissionData);
        // Check if user is allowed to transfer this object
        if (!owner || !isOwnerAdminCheck(owner, userId))
            throw new CustomError(CODE.Unauthorized, 'User is not authorized to transfer this object', { code: genErrorCode('0286') });
        // Check if the user is transferring to themselves
        const toValidator = getValidator(to.__typename, 'Transfer.request-validator');
        const toPrismaDelegate = getDelegate(to.__typename, prisma, 'Transfer.request-validator');
        const toPermissionData = await toPrismaDelegate.findUnique({
            where: { id: to.id },
            select: toValidator.permissionsSelect(userId),
        });
        const isAdmin = toPermissionData && isOwnerAdminCheck(toValidator.owner(toPermissionData), userId)
        // If so, return true. NOTE: What called this function should handle the rest
        if (isAdmin) return true
        // Create transfer request
        const request = await prisma.transfer.create({
            data: {
                fromUser: owner.User ? { connect: { id: owner.User.id } } : undefined,
                fromOrganization: owner.Organization ? { connect: { id: owner.Organization.id } } : undefined,
                [TransferableFieldMap[object.__typename]]: { connect: { id: object.id } },
                toUser: to.__typename === 'User' ? { connect: { id: to.id } } : undefined,
                toOrganization: to.__typename === 'Organization' ? { connect: { id: to.id } } : undefined,
                status: 'Pending',
                message,
            }
        });
        // Notify user/org that is receiving the object
        const pushNotification = Notify(prisma).pushTransferRequest(request.id, object.__typename);
        if (to.__typename === 'User') await pushNotification.toUser(to.id);
        else await pushNotification.toOrganization(to.id);
        // Return false
        return false
    },
    /**
     * Cancels a transfer request that you initiated
     * @param transferId The ID of the transfer request
     * @param userId The user who is cancelling the transfer request
     */
    cancel: async (transferId: string, userId: string) => {
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
            throw new CustomError(CODE.InvalidArgs, 'Transfer request does not exist', { code: genErrorCode('0293') });
        if (transfer.status === 'Accepted')
            throw new CustomError(CODE.InvalidArgs, 'Transfer request has already been accepted', { code: genErrorCode('0294') });
        if (transfer.status === 'Denied')
            throw new CustomError(CODE.InvalidArgs, 'Transfer request has already been rejected', { code: genErrorCode('0295') });
        // Make sure user is the owner of the transfer request
        if (transfer.fromOrganizationId) {
            const validator = getValidator('Organization', 'Transfer.cancel');
            const permissionData = await prisma.organization.findUnique({
                where: { id: transfer.fromOrganizationId },
                select: validator.permissionsSelect(userId),
            });
            if (!permissionData || !isOwnerAdminCheck(validator.owner(permissionData), userId))
                throw new CustomError(CODE.Unauthorized, 'User is not authorized to cancel this transfer request', { code: genErrorCode('0300') });
        } else if (transfer.fromUserId !== userId) {
            throw new CustomError(CODE.Unauthorized, 'User is not authorized to cancel this transfer request', { code: genErrorCode('0301') });
        }
        // Delete transfer request
        await prisma.transfer.delete({
            where: { id: transferId },
        });
    },
    /**
     * Accepts a transfer request
     * @param transferId The ID of the transfer request
     * @param userId The user who is accepting the transfer request
     */
    accept: async (transferId: string, userId: string) => {
        // Find the transfer request
        const transfer = await prisma.transfer.findUnique({
            where: { id: transferId },
        });
        // Make sure transfer exists, and is not accepted or rejected
        if (!transfer)
            throw new CustomError(CODE.InvalidArgs, 'Transfer request does not exist', { code: genErrorCode('0287') });
        if (transfer.status === 'Accepted')
            throw new CustomError(CODE.InvalidArgs, 'Transfer request has already been accepted', { code: genErrorCode('0288') });
        if (transfer.status === 'Denied')
            throw new CustomError(CODE.InvalidArgs, 'Transfer request has already been rejected', { code: genErrorCode('0289') });
        // Make sure transfer is going to you or an organization you can control
        if (transfer.toOrganizationId) {
            const validator = getValidator('Organization', 'Transfer.accept');
            const permissionData = await prisma.organization.findUnique({
                where: { id: transfer.toOrganizationId },
                select: validator.permissionsSelect(userId),
            });
            if (!permissionData || !isOwnerAdminCheck(validator.owner(permissionData), userId))
                throw new CustomError(CODE.Unauthorized, 'Not authorized to accept this transfer request', { code: genErrorCode('0302') });
        } else if (transfer.toUserId !== userId) {
            throw new CustomError(CODE.Unauthorized, 'Not authorized to accept this transfer request', { code: genErrorCode('0303') });
        }
        // Transfer object, then mark transfer request as accepted
        // Find the object type, based on which relation is not null
        const typeField = ['apiId', 'noteId', 'projectId', 'routineId', 'smartContractId', 'standardId'].find((field) => transfer[field] !== null);
        if (!typeField)
            throw new CustomError(CODE.InternalError, 'Transfer request is missing a relation', { code: genErrorCode('0290') });
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
        const pushNotification = Notify(prisma).pushTransferAccepted(asdf, transferId, type);
        if (transfer.fromUserId) await pushNotification.toUser(transfer.fromUserId);
        else await pushNotification.toOrganization(transfer.fromOrganizationId as string);
    },
    /**
     * Rejects a transfer request
     * @param transferId The ID of the transfer request
     * @param userId The user who is rejecting the transfer request
     * @param reason The reason for rejecting the transfer request
     */
    reject: async (transferId: string, userId: string, reason?: string) => {
        // Find the transfer request
        const transfer = await prisma.transfer.findUnique({
            where: { id: transferId },
        });
        // Make sure transfer exists, and is not already accepted or rejected
        if (!transfer)
            throw new CustomError(CODE.InvalidArgs, 'Transfer request does not exist', { code: genErrorCode('0290') });
        if (transfer.status === 'Accepted')
            throw new CustomError(CODE.InvalidArgs, 'You already accepted this transfer', { code: genErrorCode('0291') });
        if (transfer.status === 'Denied')
            throw new CustomError(CODE.InvalidArgs, 'Transfer request has already been rejected', { code: genErrorCode('0292') });
        // Make sure transfer is going to you or an organization you can control
        if (transfer.toOrganizationId) {
            const validator = getValidator('Organization', 'Transfer.reject');
            const permissionData = await prisma.organization.findUnique({
                where: { id: transfer.toOrganizationId },
                select: validator.permissionsSelect(userId),
            });
            if (!permissionData || !isOwnerAdminCheck(validator.owner(permissionData), userId))
                throw new CustomError(CODE.Unauthorized, 'Not authorized to reject this transfer request', { code: genErrorCode('0312') });
        } else if (transfer.toUserId !== userId) {
            throw new CustomError(CODE.Unauthorized, 'Not authorized to reject this transfer request', { code: genErrorCode('0313') });
        }
        // Deny the transfer
        await prisma.transfer.update({
            where: { id: transferId },
            data: {
                status: 'Denied',
                denyReason: reason,
            }
        });
        // Notify user/org that sent the transfer request
        const pushNotification = Notify(prisma).pushTransferRejected(asdf, transferId);
        if (transfer.fromUserId) await pushNotification.toUser(transfer.fromUserId);
        else await pushNotification.toOrganization(transfer.fromOrganizationId as string);
    },
})