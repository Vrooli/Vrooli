import { CODE } from "@shared/consts";
import { CustomError, genErrorCode } from "../../events";
import { onlyValidIds } from "../builder";
import { PermissionsCheckProps } from "./types";

interface ValidateHelperData {
    id: string,
    createdByUserId?: string | null | undefined,
    createdByOrganizationId?: string | null | undefined,
    projectId?: string | null | undefined,
    userId?: string | null | undefined,
    organizationId?: string | null | undefined,
}

/**
 * Validates that the user has permission to create/update/delete a project, routine or standard. 
 * Checks user/organization, project, and version. 
 * @returns True if the user has correct permisssions
 */
export async function permissionsCheck<Create extends { [x: string]: any }, Update extends { [x: string]: any}, PermissionObject>({
    actions,
    objectType,
    objectIds,
    prisma,
    userId,
}: PermissionsCheckProps<Create, Update, PermissionObject>): Promise<boolean> {
    /**
     * Helper for validating ownership of objects. 
     * Throws an error if the user does not have permission to create/update/delete the object
     */
    const validateHelper = async (data: ValidateHelperData[], isExisting: boolean): Promise<void> => {
        // Collect IDs by object type
        const userIds = onlyValidIds([...data.map(x => x.createdByUserId), ...data.map(x => x.userId)]);
        const organizationIds = onlyValidIds([...data.map(x => x.createdByOrganizationId), ...data.map(x => x.organizationId)]);
        // For projects, only need to check if isExisting is true. For existing 
        // data, we only check user/organization ownership in case permissions 
        // get messed up
        const projectIds = isExisting ? [] : onlyValidIds(data.map(x => x.projectId));
        // If any userId is not yours, throw error
        if (userIds.some(x => x !== userId)) {
            throw new CustomError(CODE.Unauthorized, 'User permissions invalid', { code: genErrorCode('0257') });
        }
        // Check organizations using roles
        const roles = await OrganizationModel.query().hasRole(prisma, userId, organizationIds);
        // If any role is undefined, the user is not authorized
        if (roles.some(x => x === undefined)) {
            throw new CustomError(CODE.Unauthorized, 'Organization permissions invalid', { code: genErrorCode('0258') });
        }
        // Check projects using projects' user/organization ownership
        if (projectIds.length > 0) {
            const projects = await prisma.project.findMany({
                where: {
                    id: { in: projectIds },
                    ...ProjectModel.permissions().ownershipQuery(userId),
                },
            })
            if (projects.length !== projectIds.length) {
                throw new CustomError(CODE.Unauthorized, 'Project permissions invalid', { code: genErrorCode('0259') });
            }
        }
    }
    /**
     * Helper for querying existing objects to validate ownership
     */
    const queryHelper = async (ids: string[]): Promise<ValidateHelperData[]> => {
        if (objectType === 'Project') {
            return await prisma.project.findMany({
                where: { id: { in: ids } },
                select: { id: true, userId: true, organizationId: true },
            });
        }
        else if (objectType === 'Routine') {
            return await prisma.routine.findMany({
                where: { id: { in: ids } },
                select: { id: true, userId: true, organizationId: true, projectId: true },
            });
        }
        else {
            return await prisma.standard.findMany({
                where: { id: { in: ids } },
                select: { id: true, createdByUserId: true, createdByOrganizationId: true },
            });
        }
    }
    // Validate createMany
    if (createMany) {
        await validateHelper(createMany, false);
    }
    // Validate updateMany
    if (updateMany) {
        await validateHelper(updateMany.map(u => u.data), false);
        const existingObjects = await queryHelper(updateMany.map(u => u.where.id));
        await validateHelper(existingObjects, true);
    }
    // Validate deleteMany
    if (deleteMany) {
        const existingObjects = await queryHelper(deleteMany);
        await validateHelper(existingObjects, true);
    }
}