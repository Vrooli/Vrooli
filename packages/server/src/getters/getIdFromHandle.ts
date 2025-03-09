import { ModelType } from "@local/shared";
import { DbProvider } from "../db/provider.js";

/**
 * Finds the id of an object from its handle
 * @returns The id of the object
 */
export async function getIdFromHandle({
    handle,
    objectType,
}: {
    handle: string;
    objectType: `${ModelType}`,
}): Promise<string | undefined> {
    const where = { handle };
    const select = { id: true };
    const query = { where, select };
    const id = objectType === "Team" ? await DbProvider.get().team.findFirst(query) :
        objectType === "Project" ? await DbProvider.get().project.findFirst(query) :
            await DbProvider.get().user.findFirst(query);
    return id?.id;
}
