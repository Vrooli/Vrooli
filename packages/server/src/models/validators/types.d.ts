import { PrismaType } from "../../types";
import { GraphQLModelType, ModelLogic } from "../types";

export interface MaxObjectsCheckProps<GraphQLCreate extends { [x: string]: any }, GraphQLUpdate extends { [x: string]: any }> {
    createMany?: GraphQLCreate[] | null | undefined,
    deleteMany?: string[] | null | undefined,
    objectType: GraphQLModelType,
    prisma: PrismaType,
    updateMany?: { where: { [x: string]: any }, data: GraphQLUpdate }[] | null | undefined,
    userId: string,
}

export interface PermissionsCheckProps<Create extends { [x: string]: any }, Update extends { [x: string]: any}, PermissionObject> {
    /**
     * Array of actions to check for
     */
    actions: string[];
    objectType: GraphQLModelType,
    objectIds: string[],
    prisma: PrismaType;
    userId: string | null;
}