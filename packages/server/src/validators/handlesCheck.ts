import { prismaInstance } from "../db/instance";
import { CustomError } from "../events/error";
import { ModelMap } from "../models/base";
import { hasProfanity } from "../utils/censor";

/** 
 * Handles that are not allowed to be used, regardless if they are in the database.
 * These URL conflicts, object types, and handles reserved for internal use.
 */
const RESERVED_HANDLES = [
    "abuse",
    "add",
    "api",
    "admin",
    "administrator",
    "assistant",
    "billing",
    "bot",
    "careers",
    "community",
    "compliance",
    "contact",
    "customerservice",
    "customer_service",
    "create",
    "delete",
    "dev",
    "developer",
    "edit",
    "feedback",
    "finance",
    "find",
    "help",
    "hr",
    "humanresources",
    "human_resources",
    "info",
    "investor",
    "legal",
    "management",
    "manager",
    "marketing",
    "meeting",
    "mod",
    "moderator",
    "note",
    "official",
    "organization",
    "partners",
    "press",
    "privacy",
    "profile",
    "project",
    "question",
    "root",
    "routine",
    "run",
    "sales",
    "schedule",
    "security",
    "service",
    "smart_contract",
    "smartcontract",
    "standard",
    "status",
    "support",
    "system",
    "tag",
    "team",
    "terms",
    "test",
    "update",
    "upsert",
    "user",
    "valyxa",
    "valyxaofficial",
    "valyxa_official",
    "vrooli",
    "vrooliofficial",
    "vrooli_official",
    "webmaster",
];

/**
 * Verifies that handles are available 
 * @param forType The type of object to check handles for
 * @param Create Handle and id pairs for new objects
 * @param Update Handle and id pairs for updated objects
 * @param languages Preferred languages for error messages
 */
export const handlesCheck = async (
    forType: "User" | "Organization" | "Project",
    Create: { input: { id: string, handle?: string | null | undefined } }[],
    Update: { input: { id: string, handle?: string | null | undefined } }[],
    languages: string[],
): Promise<void> => {
    // Filter out empty handles from createList and updateList
    const filteredCreateList = Create.filter(x => x.input.handle).map(x => x.input) as { id: string, handle: string }[];
    const filteredUpdateList = Update.filter(x => x.input.handle).map(x => x.input) as { id: string, handle: string }[];

    // Find all existing handles that match the handles in createList and updateList.
    // There should be none, unless some of the updates are changing the existing handles to something else.
    const { delegate } = ModelMap.getLogic(["delegate"], forType);
    const existingHandles = await delegate(prismaInstance).findMany({
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
        if (Update.find(x => x.input.id === id && x.input.handle !== handle)) {
            handleUsage[handle]--;
        }
        // Check if a handle is taken
        if (handleUsage[handle] > 0) {
            throw new CustomError("0019", "HandleTaken", languages);
        }
    }
};
