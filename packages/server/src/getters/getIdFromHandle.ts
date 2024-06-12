import { GqlModelType } from "@local/shared";
import { prismaInstance } from "../db/instance";

/**
 * Finds the id of an object from its handle
 * @returns The id of the object
 */
export async function getIdFromHandle({
    handle,
    objectType,
}: {
    handle: string;
    objectType: `${GqlModelType}`,
}): Promise<string | undefined> {
    const where = { handle };
    const select = { id: true };
    const query = { where, select };
    const id = objectType === "Team" ? await prismaInstance.team.findFirst(query) :
        objectType === "Project" ? await prismaInstance.project.findFirst(query) :
            await prismaInstance.user.findFirst(query);
    return id?.id;
}
