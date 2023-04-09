import { CustomError } from "../events";
import { PrismaType } from "../types";
import { hasProfanity } from "../utils/censor";

/**
* Maps GqlModelType to wallet relationship field
*/
const walletOwnerMap = {
    User: 'userId',
    Organization: 'organizationId',
    Project: 'projectId',
} as const

/**
 * Verifies that one or more handles are owned by a wallet, that is owned by an object. 
 * Throws error on failure. 
 * @param prisma Prisma client
 * @param forType The type of wallet-owning object to check
 * @param list The list of ids and handles to check
 * @param languages Preferred languages for error messages
 * @param checkProfanity Whether to check for profanity in the handle
 */
export const handlesCheck = async (
    prisma: PrismaType,
    forType: 'User' | 'Organization' | 'Project',
    list: { id: string, handle?: string | null | undefined }[],
    languages: string[],
    checkProfanity: boolean = true
): Promise<void> => {
    // Filter out empty handles
    const filtered = list.filter(x => x.handle) as { id: string, handle: string }[];
    // Query the database for wallets
    const wallets = await prisma.wallet.findMany({
        where: { [walletOwnerMap[forType]]: { in: filtered.map(x => x.id) } },
        select: {
            handles: {
                select: {
                    handle: true,
                }
            }
        }
    });
    // Check that each handle is owned by the wallet
    for (const { id, handle } of filtered) {
        const wallet = wallets.find(w => w[walletOwnerMap[forType]] === id);
        if (!wallet) {
            throw new CustomError('0019', 'ErrorUnknown', languages);
        }
        const found = Boolean(wallet.handles.find(h => h.handle === handle));   
        if (!found) {
            throw new CustomError('0374', 'ErrorUnknown', languages);
        }
        if (checkProfanity && hasProfanity(handle)) {
            throw new CustomError('0375', 'BannedWord', languages);
        }
    }
}