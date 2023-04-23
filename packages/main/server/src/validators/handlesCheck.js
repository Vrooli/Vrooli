import { CustomError } from "../events";
import { hasProfanity } from "../utils/censor";
const walletOwnerMap = {
    User: "userId",
    Organization: "organizationId",
    Project: "projectId",
};
export const handlesCheck = async (prisma, forType, list, languages, checkProfanity = true) => {
    const filtered = list.filter(x => x.handle);
    const wallets = await prisma.wallet.findMany({
        where: { [walletOwnerMap[forType]]: { in: filtered.map(x => x.id) } },
        select: {
            handles: {
                select: {
                    handle: true,
                },
            },
        },
    });
    for (const { id, handle } of filtered) {
        const wallet = wallets.find(w => w[walletOwnerMap[forType]] === id);
        if (!wallet) {
            throw new CustomError("0019", "ErrorUnknown", languages);
        }
        const found = Boolean(wallet.handles.find(h => h.handle === handle));
        if (!found) {
            throw new CustomError("0374", "ErrorUnknown", languages);
        }
        if (checkProfanity && hasProfanity(handle)) {
            throw new CustomError("0375", "BannedWord", languages);
        }
    }
};
//# sourceMappingURL=handlesCheck.js.map