import { PrismaType } from "../../types";
import { GraphQLModelType } from "../types";

export interface GetIdFromHandleProps {
    handle: string;
    objectType: 'Organization' | 'Project' | 'User';
    prisma: PrismaType;
}

export interface GetLatestVersionProps {
    includeIncomplete?: boolean,
    objectType: 'Api' | 'Note' | 'Routine' | 'SmartContract' | 'Standard',
    prisma: PrismaType,
    versionGroupId: string,
}

export interface GetPermissionsProps {
    objectType: GraphQLModelType;
    ids: string[];
    prisma: PrismaType;
    userId: string | null;
}

export type QueryAction = 'Connect' | 'Create' | 'Delete' | 'Disconnect' | 'Read' | 'Update';