import { GqlModelType } from "@shared/consts";
import { PrismaType } from "../types";

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
    objectType: GqlModelType;
    ids: string[];
    prisma: PrismaType;
    userId: string | null;
}

export type QueryAction = 'Connect' | 'Create' | 'Delete' | 'Disconnect' | 'Read' | 'Update';