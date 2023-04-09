import { GqlModelType } from "@shared/consts";
import { PrismaType } from "../types";

/**
 * Finds the id of an object from its handle
 * @returns The id of the object
 */
export async function getIdFromHandle({
    handle,
    objectType,
    prisma,
}: {
    handle: string;
    objectType: `${GqlModelType}`,
    prisma: PrismaType,
}): Promise<string | undefined> {
    const where = { handle };
    const select = { id: true };
    const query = { where, select };
    const id = objectType === 'Organization' ? await prisma.organization.findFirst(query) :
        objectType === 'Project' ? await prisma.project.findFirst(query) :
            await prisma.user.findFirst(query);
    return id?.id;
}