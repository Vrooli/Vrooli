import { CustomError } from "../events";
import { getLogic } from "../getters";
import { PrismaType } from "../types";
import { hasProfanity } from "../utils/censor";

/** 
 * Handles that are not allowed to be used, regardless if they are in the database.
 * These URL conflicts, object types, and handles reserved for internal use.
 */
const RESERVED_HANDLES = [
    "api",
    "admin",
    "bot",
    "help",
    "note",
    "organization",
    "profile",
    "project",
    "question",
    "routine",
    "smart_contract",
    "smartcontract",
    "standard",
    "support",
    "valyxa",
    "valyxaofficial",
    "valyxa_official",
    "vrooli",
    "vrooliofficial",
    "vrooli_official",
];

/**
 * Verifies that handles are available 
 * @param prisma Prisma client
 * @param forType The type of object to check handles for
 * @param createList Handle and id pairs for new objects
 * @param updateList Handle and id pairs for updated objects
 * @param languages Preferred languages for error messages
 */
export const handlesCheck = async (
    prisma: PrismaType,
    forType: "User" | "Organization" | "Project",
    createList: { id: string, handle?: string | null | undefined }[],
    updateList: { id: string, handle?: string | null | undefined }[],
    languages: string[],
): Promise<void> => {
    // Filter out empty handles from createList and updateList
    const filteredCreateList = createList.filter(x => x.handle) as { id: string, handle: string }[];
    const filteredUpdateList = updateList.filter(x => x.handle) as { id: string, handle: string }[];

    // Find all existing handles that match the handles in createList and updateList.
    // There should be none, unless some of the updates are changing the existing handles to something else.
    const { delegate } = getLogic(["delegate"], forType, languages, "handlesCheck");
    const existingHandles = await delegate(prisma).findMany({
        where: { handle: { in: [...filteredCreateList, ...filteredUpdateList].map(x => x.handle) } },
        select: { id: true, handle: true },
    });

    // Track how many times each handle in createList and updateList is used. 
    // This should be one for each handle
    const handleUsage: { [key: string]: number } = {};
    for (const { handle } of [...filteredCreateList, ...filteredUpdateList]) {
        if (handle) {
            handleUsage[handle] = (handleUsage[handle] || 0) + 1;
            // If there is more than one use of a handle, throw an error
            if (handleUsage[handle] > 1) {
                throw new CustomError("0019", "HandleTaken", languages);
            }
            // Also check for profanity while we're at it
            if (hasProfanity(handle)) {
                throw new CustomError("0374", "BannedWord", languages);
            }
            // Also check for reserved handles while we're at it
            if (RESERVED_HANDLES.includes(handle.toLowerCase())) {
                throw new CustomError("0375", "HandleTaken", languages);
            }
        }
    }

    // Loop through existing handles
    for (const { id, handle } of existingHandles) {
        // Decrease the count if the handle is being changed to something else
        if (updateList.find(x => x.id === id && x.handle !== handle)) {
            handleUsage[handle]--;
        }
        // Check if a handle is taken
        if (handleUsage[handle] > 0) {
            throw new CustomError("0019", "HandleTaken", languages);
        }
    }
};
