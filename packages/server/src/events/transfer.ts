import { getDelegate, getDisplay, getValidator } from "../getters";
import { Notify } from "../notify";
import { SessionUser } from "../schema/types";
import { PrismaType } from "../types";
import { isOwnerAdminCheck } from "../validators";
import { CustomError } from "./error";

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
     * @param userData The session data of the user who is transferring the object
     * @param message Optional message to include with the transfer request
     * @returns True if transfer is complete (i.e. the user is transferring to an organization where they have the correct permissions), 
     * false if the transfer is pending
     */
    request: async (
        to: { __typename: 'Organization' | 'User', id: string },
        object: { __typename: TransferableObjects, id: string },
        userData: SessionUser,
        message?: string,
    ): Promise<boolean> => {
        // Find the object and its owner
        const validator = getValidator(object.__typename, userData.languages, 'Transfer.request-object');
        const prismaDelegate = getDelegate(object.__typename, prisma, userData.languages, 'Transfer.request-object');
        const permissionData = await prismaDelegate.findUnique({
            where: { id: object.id },
            select: validator.permissionsSelect,
        });
        const owner = permissionData && validator.owner(permissionData);
        // Check if user is allowed to transfer this object
        if (!owner || !isOwnerAdminCheck(owner, userData.id))
            throw new CustomError('0286', 'NotAuthorizedToTransfer', userData.languages);
        // Check if the user is transferring to themselves
        const toValidator = getValidator(to.__typename, userData.languages, 'Transfer.request-validator');
        const toPrismaDelegate = getDelegate(to.__typename, prisma, userData.languages, 'Transfer.request-validator');
        const toPermissionData = await toPrismaDelegate.findUnique({
            where: { id: to.id },
            select: toValidator.permissionsSelect(userData.id, userData.languages),
        });
        const isAdmin = toPermissionData && isOwnerAdminCheck(toValidator.owner(toPermissionData), userData.id)
        // If so, return true. NOTE: What called this function should handle the rest
        if (isAdmin) return true;
        // Find a user-readable label for the object
        const requesteeLanguages = (owner.Organization?.languages || owner.User?.languages).map(l => l.language);
        const display = getDisplay(object.__typename, userData.languages, 'Transfer.request');
        const objectTitle = (await display.labels(prisma, [{ id: object.id, languages: requesteeLanguages }]))[0];
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
                objectTitle,
            }
        });
        // Notify user/org that is receiving the object
        const pushNotification = Notify(prisma, userData.languages).pushTransferRequest(request.id, object.__typename);
        if (to.__typename === 'User') await pushNotification.toUser(to.id);
        else await pushNotification.toOrganization(to.id, userData.id);
        // Return false
        return false
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
        const pushNotification = Notify(prisma, userData.languages).pushTransferAccepted(transfer.objectTitle, transferId, type);
        if (transfer.fromUserId) await pushNotification.toUser(transfer.fromUserId);
        else await pushNotification.toOrganization(transfer.fromOrganizationId as string, userData.id);
    },
    /**
     * Rejects a transfer request
     * @param transferId The ID of the transfer request
     * @param userData The session data of the user who is rejecting the transfer request
     * @param reason Optional reason for rejecting the transfer request
     */
    reject: async (transferId: string, userData: SessionUser, reason?: string) => {
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
        const pushNotification = Notify(prisma, userData.languages).pushTransferRejected(transfer.objectTitle, type, transferId);
        if (transfer.fromUserId) await pushNotification.toUser(transfer.fromUserId);
        else await pushNotification.toOrganization(transfer.fromOrganizationId as string, userData.id);
    },
})