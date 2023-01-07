import { CustomError } from "../events";
import { PrismaType } from "../types";
import { hasProfanity } from "./censor";

/**
* Maps GqlModelType to wallet relationship field
*/
const walletOwnerMap = {
    User: 'userId',
    Organization: 'organizationId',
    Project: 'projectId',
}

/**
 * Verify that a handle is owned by a wallet, that is owned by an object. 
 * Throws error on failure. 
 * Allows checks for profanity, cause why not
 * @params for The type of object that the wallet is owned by (i.e. Project, Organization, User)
 * @params forId The ID of the object that the wallet is owned by
 * @params handle The handle to verify
 * @params languages Preferred languages for error messages
 */
export const verifyHandle = async (
    prisma: PrismaType,
    forType: 'User' | 'Organization' | 'Project',
    forId: string,
    handle: string | null | undefined,
    languages: string[]
): Promise<void> => {
    if (!handle) return;
    // Check that handle belongs to one of user's wallets
    const wallets = await prisma.wallet.findMany({
        where: { [walletOwnerMap[forType]]: forId },
        select: {
            handles: {
                select: {
                    handle: true,
                }
            }
        }
    });
    const found = Boolean(wallets.find(w => w.handles.find(h => h.handle === handle)));
    if (!found)
        throw new CustomError('0019', 'ErrorUnknown', languages);
    // Check for censored words
    if (hasProfanity(handle))
        throw new CustomError('0120', 'BannedWord', languages);
}