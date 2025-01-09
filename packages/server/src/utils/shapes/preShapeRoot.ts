import { exists, ModelType, SessionUser } from "@local/shared";
import { PrismaDelegate } from "../../builders/types";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { ModelMap } from "../../models/base";
import { transfer } from "../../models/base/transfer";

type HasCompleteVersionData = {
    hasCompleteVersion: boolean,
    completedAt: Date | null,
};

type ObjectTriggerData = {
    hasCompleteAndPublic: boolean,
    wasCompleteAndPublic: boolean,
    hasBeenTransferred: boolean,
    hasParent: boolean,
    owner: {
        id: string,
        __typename: "User" | "Team",
    }
};

type PreShapeRootParams = {
    Create: {
        input: {
            id: string,
            isPrivate?: boolean | null | undefined,
            ownedByTeamConnect?: string | null | undefined,
            ownedByUserConnect?: string | null | undefined,
            parentConnect?: string | null | undefined,
            versionsCreate?: {
                id: string,
                isComplete?: boolean | null | undefined,
                isPrivate?: boolean | null | undefined,
            }[] | null | undefined,
        }
    }[],
    Update: {
        input: {
            id: string,
            isPrivate?: boolean | null | undefined,
            ownedByTeamConnect?: string | null | undefined,
            ownedByUserConnect?: string | null | undefined,
            parentConnect?: string | null | undefined,
            versionsCreate?: {
                id: string,
                isComplete?: boolean | null | undefined,
                isPrivate?: boolean | null | undefined,
            }[] | null | undefined,
            versionsUpdate?: {
                id: string,
                isComplete?: boolean | null | undefined,
                isPrivate?: boolean | null | undefined,
            }[] | null | undefined,
            versionsDelete?: string[] | null | undefined,
        }
    }[],
    Delete: {
        input: string;
    }[],
    objectType: ModelType | `${ModelType}`,
    userData: SessionUser,
};

export type PreShapeRootResult = {
    versionMap: Record<string, HasCompleteVersionData>,
    triggerMap: Record<string, { hasCompleteAndPublic: boolean, wasCompleteAndPublic: boolean }>,
    transferMap: Record<string, boolean>,
};

const originalDataSelect = {
    id: true,
    hasBeenTransferred: true,
    isPrivate: true,
    ownedByTeam: { select: { id: true } },
    ownedByUser: { select: { id: true } },
    parent: { select: { id: true } },
    versions: {
        select: {
            id: true,
            isComplete: true,
            isPrivate: true,
        },
    },
};

/**
 * Used in mutate.shape.pre of root objects. Has three purposes:
 * 1. Calculate hasCompleteVersion flag and completedAt date to update object in database)
 * 2. Calculate data for objectCreated/Updated/Deleted trigger
 * 3. Determine which creates/updates require a transfer request
 * @returns maps for version, trigger, and transfer data
 */
export async function preShapeRoot({
    Create,
    Update,
    Delete,
    objectType,
    userData,
}: PreShapeRootParams): Promise<PreShapeRootResult> {
    // Get db table
    const { dbTable } = ModelMap.getLogic(["dbTable"], objectType);
    // Calculate hasCompleteVersion and hasCompleteAndPublic version flags
    const versionMap: Record<string, HasCompleteVersionData> = {};
    const triggerMap: Record<string, ObjectTriggerData> = {};
    const transferMap: Record<string, boolean> = {};
    // For createList (very simple)
    for (const { input } of Create) {
        const hasCompleteVersion = input.versionsCreate?.some(v => v.isComplete) ?? false;
        versionMap[input.id] = {
            hasCompleteVersion,
            completedAt: hasCompleteVersion ? new Date() : null,
        };
        triggerMap[input.id] = {
            wasCompleteAndPublic: true, // Doesn't matter
            hasCompleteAndPublic: !input.isPrivate && (input.versionsCreate?.some(v => v.isComplete && !v.isPrivate) ?? false),
            hasBeenTransferred: false, // Doesn't matter
            hasParent: typeof input.parentConnect === "string",
            owner: {
                id: (input.ownedByUserConnect ?? input.ownedByTeamConnect) as string,
                __typename: input.ownedByUserConnect ? "User" : "Team",
            },
        };
    }
    // For updateList (much more complicated)
    if (Update.length > 0) {
        // Find original data
        const originalData = await (prismaInstance[dbTable] as PrismaDelegate).findMany({
            where: { id: { in: Update.map(u => u.input.id) } },
            select: originalDataSelect,
        });
        // Loop through updates
        for (const { input } of Update) {
            // Find original
            const original = originalData.find(r => r.id === input.id);
            if (!original) throw new CustomError("0412", "InternalError", { id: input?.id });
            const isRootPrivate = input.isPrivate ?? original.isPrivate;
            // Convert original versions to map for easy lookup
            const updatedWithOriginal = original.versions.reduce((acc, v) => ({ ...acc, [v.id]: v }), {} as Record<string, any>);
            // Combine updated versions with original versions
            if (Array.isArray(input.versionsUpdate)) {
                for (const v of input.versionsUpdate) {
                    updatedWithOriginal[v.id] = {
                        ...updatedWithOriginal[v.id],
                        ...v,
                    };
                }
            }
            // Combine new, updated, and original versions. Then remove deleting versions
            const allVersions: any[] = Object.values(updatedWithOriginal).concat(input.versionsCreate ?? []);
            const versions = allVersions.filter(v => !input.versionsDelete?.includes(v.id));
            // Calculate flags
            const hasCompleteVersion = versions.some(v => v.isComplete);
            versionMap[input.id] = {
                hasCompleteVersion,
                completedAt: hasCompleteVersion ? new Date() : null,
            };
            triggerMap[input.id] = {
                wasCompleteAndPublic: !original.isPrivate && original.versions.some(v => v.isComplete && !v.isPrivate),
                hasCompleteAndPublic: !isRootPrivate && versions.some(v => v.isComplete && !v.isPrivate),
                hasBeenTransferred: original.hasBeenTransferred,
                hasParent: exists(original.parent),
                // TODO owner might be changed here depending on how triggers are implemented.
                // For now, using original owner
                owner: {
                    id: original.ownedByUser?.id ?? original.ownedByTeam?.id,
                    __typename: original.ownedByUser ? "User" : "Team",
                },
            };
        }
    }
    // For deleteList (fairly simple)
    if (Delete.length > 0) {
        // Find original data
        const originalData = await (prismaInstance[dbTable] as PrismaDelegate).findMany({
            where: { id: { in: Delete.map(d => d.input) } },
            select: originalDataSelect,
        });
        // Loop through deletes
        for (const { input: id } of Delete) {
            // Find original
            const original = originalData.find(r => r.id === id);
            if (!original) throw new CustomError("0413", "InternalError", { id });
            triggerMap[id] = {
                wasCompleteAndPublic: !original.isPrivate && original.versions.some(v => v.isComplete && !v.isPrivate),
                hasCompleteAndPublic: true, // Doesn't matter
                hasBeenTransferred: original.hasBeenTransferred,
                hasParent: exists(original.parent),
                // TODO owner might be changed here depending on how triggers are implemented.
                // For now, using original owner
                owner: {
                    id: original.ownedByUser?.id ?? original.ownedByTeam?.id,
                    __typename: original.ownedByUser ? "User" : "Team",
                },
            };
        }
    }
    // Finally determine which creates/updates require a transfer request
    // Get create and update owners from triggerMap and map them to object IDs. Make sure to filter out deleteList owners
    const ownersEntries = Object.entries(triggerMap)
        .filter(([id]) => !Delete.some(d => d.input === id))
        .map(([id, data]) => [id, data.owner]);
    const ownersMap = Object.fromEntries(ownersEntries);
    const requireTransfers = await transfer().checkTransferRequests(Object.values(ownersMap), userData);
    for (let i = 0; i < requireTransfers.length; i++) {
        const id = Object.keys(ownersMap)[i];
        transferMap[id] = requireTransfers[i];
    }
    return { versionMap, triggerMap, transferMap };
}
