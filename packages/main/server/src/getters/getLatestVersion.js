import { isOfType } from "@local/utils";
import { CustomError } from "../events";
import { ObjectMap } from "../models";
export async function getLatestVersion({ includeIncomplete = true, objectType, prisma, idRoot, handleRoot, }) {
    if (!idRoot && !handleRoot)
        throw new CustomError("0371", "InvalidArgs", ["en"], { objectType });
    const handleableObjects = ["ProjectVersion"];
    if (handleRoot && !handleableObjects.includes(objectType))
        throw new CustomError("0372", "InvalidArgs", ["en"], { objectType, handleRoot });
    const rootSelectQuery = idRoot ? { id: idRoot } : { handle: handleRoot };
    const model = ObjectMap[objectType];
    if (!model)
        throw new CustomError("0373", "InternalError", ["en"], { objectType });
    if (isOfType({ __typename: objectType }, "Api", "Note")) {
        const query = {
            where: { root: rootSelectQuery },
            orderBy: { versionIndex: "desc" },
            select: { id: true },
        };
        const latestVersion = await model.delegate(prisma).findFirst(query);
        return latestVersion?.id;
    }
    else {
        const query = {
            where: { root: rootSelectQuery, isComplete: includeIncomplete ? undefined : true },
            orderBy: { versionIndex: "desc" },
            select: { id: true },
        };
        const latestVersion = await model.delegate(prisma).findFirst(query);
        return latestVersion?.id;
    }
}
//# sourceMappingURL=getLatestVersion.js.map