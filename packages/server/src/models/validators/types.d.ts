import { PrismaType } from "../../types";
import { BasePermissions, GraphQLModelType, ModelLogic } from "../types";

export interface MaxObjectsCheckProps<GraphQLCreate extends { [x: string]: any }, GraphQLUpdate extends { [x: string]: any }> {
    createMany?: GraphQLCreate[] | null | undefined,
    deleteMany?: string[] | null | undefined,
    objectType: GraphQLModelType,
    prisma: PrismaType,
    updateMany?: { where: { [x: string]: any }, data: GraphQLUpdate }[] | null | undefined,
    userId: string,
}

export interface PermissionsCheckProps<PermissionObject extends BasePermissions> {
    actions: PermissionType[];
    permissions: PermissionObject[];
}

export type PermissionType = 'Create' | 'Read' | 'Update' | 'Delete' | 'Fork' | 'Report' | 'Run';