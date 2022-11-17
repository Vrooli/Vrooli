import { CODE } from "@shared/consts";
import { GraphQLModelType } from "../models/types";
import { getDelegate, getValidator } from "../models/utils";
import { PrismaType } from "../types";
import { CustomError } from "./error";
import { genErrorCode } from "./logger";

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
        object: { __typename: GraphQLModelType, id: string },
        userId: string,
    ): Promise<boolean> => {
        // Find the object and its owner
        const validator = getValidator(object.__typename, 'Transfer.initiate');
        const prismaDelegate = getDelegate(object.__typename, prisma, 'Transfer.initiate');
        const permissionData = await prismaDelegate.findUnique({
            where: { id: object.id },
            select: validator.permissionsSelect,
        });
        // Check if user is allowed to transfer this object
        if (!permissionData || !validator.isAdmin(permissionData, userId))
            throw new CustomError(CODE.Unauthorized, 'User is not authorized to transfer this object', { code: genErrorCode('0286') });
        // Check if the user is transferring to themselves
        fdsafdsafds
        // If so, return true
        if (asdfad) return true
        // Otherwise, object will keep original owner for now
        // Create transfer request
        asdfasdfa
        // Notify user/org that is receiving the object
        asdfasd
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
        });
        // Make sure transfer exists, and is not already accepted or rejected
        if (!transfer)
            throw new CustomError(CODE.InvalidArgs, 'Transfer request does not exist', { code: genErrorCode('0293') });
        if (transfer.status === 'Accepted')
            throw new CustomError(CODE.InvalidArgs, 'Transfer request has already been accepted', { code: genErrorCode('0294') });
        if (transfer.status === 'Denied')
            throw new CustomError(CODE.InvalidArgs, 'Transfer request has already been rejected', { code: genErrorCode('0295') });
        // Make sure user is the owner of the transfer request
        fdasfs
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
        ffdsafdas
        // Accept the transfer
        fdsafd
    },
    /**
     * Rejects a transfer request
     * @param transferId The ID of the transfer request
     * @param userId The user who is rejecting the transfer request
     * @param reason The reason for rejecting the transfer request
     */
    reject: async (transferId: string, userId: string, reason: string) => {
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
        ffdsafdas
        // Accept the transfer
        fdsafd
    },
})