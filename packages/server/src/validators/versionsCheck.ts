import { GqlModelType } from "@local/shared";
import { PrismaDelegate } from "../builders/types";
import { prismaInstance } from "../db/instance";
import { CustomError } from "../events/error";
import { ModelMap } from "../models/base";
import { SessionUserToken } from "../types";

function hasInternalField(objectType: string) {
    return [GqlModelType.RoutineVersion, GqlModelType.StandardVersion].includes(objectType as any);
}

/**
 * Checks if versions of an object type can be created, updated, or deleted.
 * Throws error on failure.
 * Requirements:
 * 1. The root object and updating versions are not soft-deleted
 * 2. Version labels are unique per root object, including existing versions in the database
 * 3. Updating versions are not marked as complete, OR the version or root is private (i.e. isPrivate = true). 
 * This helps ensure that public data is immutable, while owners have full control over private data
 */
export async function versionsCheck({
    objectType,
    Create,
    Update,
    Delete,
    userData,
}: {
    objectType: `${GqlModelType.ApiVersion
    | GqlModelType.CodeVersion
    | GqlModelType.NoteVersion
    | GqlModelType.ProjectVersion
    | GqlModelType.RoutineVersion
    | GqlModelType.StandardVersion
    }`,
    Create: {
        input: {
            id: string,
            rootConnect?: string | null | undefined,
            rootCreate?: { id: string } | null | undefined,
            versionLabel?: string | null | undefined
        }
    }[],
    Update: {
        input: {
            id: string,
            versionLabel?: string | null | undefined,
        }
    }[],
    Delete: {
        input: string;
    }[],
    userData: SessionUserToken,
}) {
    // Filter unchanged versions from create and update data
    const create = Create.filter(x => x.input.versionLabel).map(({ input }) => {
        const rootData = input.rootCreate ?? input.rootConnect;
        const rootId = typeof rootData === "string" ? rootData : rootData?.id as string;
        return {
            id: input.id,
            rootId,
            versionLabel: input.versionLabel,
        };
    });
    const update = Update.filter(x => x.input.versionLabel).map(({ input }) => ({
        id: input.id,
        versionLabel: input.versionLabel,
    }));
    // Find unique root ids from create data
    const createRootIds = create.map(x => x.rootId);
    const uniqueRootIds = [...new Set(createRootIds)];
    // Find unique version ids from update and delete data
    const updateIds = update.map(x => x.id);
    const deleteIds = Delete;
    const uniqueVersionIds = [...new Set([...updateIds, ...deleteIds])];
    // Query the database for existing data (by root)
    const rootType = objectType.replace("Version", "") as GqlModelType;
    const dbTable = ModelMap.get(rootType).dbTable;
    let existingRoots: any[];
    let where: { [key: string]: any } = {};
    let select: { [key: string]: any } = {};
    try {
        where = {
            OR: [
                { id: { in: uniqueRootIds } },
                { versions: { some: { id: { in: uniqueVersionIds } } } },
            ],
        };
        select = {
            id: true,
            hasCompleteVersion: true,
            isDeleted: true,
            // Also query isInternal if applicable
            ...(hasInternalField(objectType) && { isInternal: true }),
            isPrivate: true,
            versions: {
                select: {
                    id: true,
                    isComplete: true,
                    isDeleted: true,
                    isPrivate: true,
                    versionLabel: true,
                },
            },
        };
        existingRoots = await (prismaInstance[dbTable] as PrismaDelegate).findMany({
            where,
            select,
        });
    } catch (error) {
        throw new CustomError("0414", "InternalError", userData.languages, { error, where, select, rootType });
    }
    for (const root of existingRoots) {
        // Check 1
        // Root cannot already be deleted
        if (root.isDeleted) {
            throw new CustomError("0377", "ErrorUnknown", userData.languages);
        }
        // Updating versions cannot be deleted
        for (const version of root.versions) {
            if (updateIds.includes(version.id) && version.isDeleted) {
                throw new CustomError("0378", "ErrorUnknown", userData.languages);
            }
        }
        // Check 2
        // Group ids and labels of existing versions, which are not being deleted
        const versionIds = root.versions.filter(x => !deleteIds.includes(x.id)).map(x => x.id);
        const versionLabels = root.versions.filter(x => !deleteIds.includes(x.id)).map(x => x.versionLabel);
        // New versions cannot have the same label as existing versions
        const createLabels = create.filter(x => x.rootId === root.id).map(x => x.versionLabel);
        if (createLabels.some(x => versionLabels.includes(x))) {
            throw new CustomError("0379", "ErrorUnknown", userData.languages);
        }
        // Updating versions cannot have the same label as existing versions
        const updateLabels = update.filter(x => versionIds.includes(x.id)).map(x => x.versionLabel);
        // We must filter out updating labels from the existing labels, to support swapping
        const versionLabelsWithoutUpdate = root.versions.filter(x => !deleteIds.includes(x.id) && !updateIds.includes(x.id)).map(x => x.versionLabel);
        if (updateLabels.some(x => versionLabelsWithoutUpdate.includes(x))) {
            throw new CustomError("0380", "ErrorUnknown", userData.languages);
        }
        // Check 3
        // If the root is not private and not internal (if applicable)
        if (!root.isPrivate && !(hasInternalField(objectType) && root.isInternal)) {
            // Updating versions (which are not private) cannot be marked as complete
            for (const version of root.versions) {
                if (updateIds.includes(version.id) && !version.isPrivate && version.isComplete) {
                    throw new CustomError("0381", "ErrorUnknown", userData.languages);
                }
            }
        }
    }
}
