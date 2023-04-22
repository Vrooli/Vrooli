import { GqlModelType } from "@shared/consts";
import { isOfType } from "@shared/utils";
import { CustomError } from "../events";
import { ObjectMap } from "../models";
import { PrismaType } from "../types";

/**
 * Finds the latest version of a versioned object, using the root object's ID OR handle
 * @returns The id of the latest version
 */
export async function getLatestVersion({
    includeIncomplete = true,
    objectType,
    prisma,
    idRoot,
    handleRoot,
}: {
    includeIncomplete?: boolean,
    objectType: `${GqlModelType.ApiVersion | GqlModelType.NoteVersion | GqlModelType.ProjectVersion | GqlModelType.RoutineVersion | GqlModelType.SmartContractVersion | GqlModelType.StandardVersion}`,
    prisma: PrismaType,
    idRoot?: string | null,
    handleRoot?: string | null,
}): Promise<string | undefined> {
    // Make sure either idRoot or handleRoot is provided
    if (!idRoot && !handleRoot) throw new CustomError("0371", "InvalidArgs", ["en"], { objectType });
    // Make sure handle is only supplied for objects that have handles
    const handleableObjects = ["ProjectVersion"];
    if (handleRoot && !handleableObjects.includes(objectType)) throw new CustomError("0372", "InvalidArgs", ["en"], { objectType, handleRoot });
    // Create root select query
    const rootSelectQuery = idRoot ? { id: idRoot } : { handle: handleRoot };
    // Get model
    const model = ObjectMap[objectType];
    if (!model) throw new CustomError("0373", "InternalError", ["en"], { objectType });
    // Handle apis and notes, which don't have an "isComplete" field
    if (isOfType({ __typename: objectType }, "Api", "Note")) {
        const query = {
            where: { root: rootSelectQuery },
            orderBy: { versionIndex: "desc" as const },
            select: { id: true },
        };
        const latestVersion = await model.delegate(prisma).findFirst(query);
        return latestVersion?.id;
    }
    // Handle other objects, which do have an "isComplete" field
    else {
        const query = {
            where: { root: rootSelectQuery, isComplete: includeIncomplete ? undefined : true },
            orderBy: { versionIndex: "desc" as const },
            select: { id: true },
        };
        const latestVersion = await model.delegate(prisma).findFirst(query);
        return latestVersion?.id;
    }
}
