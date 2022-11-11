import { PrismaType } from "../../types";

export interface GetLatestVersionProps {
    includeIncomplete?: boolean,
    objectType: 'Api' | 'Note' | 'Routine' | 'SmartContract' | 'Standard',
    prisma: PrismaType,
    versionGroupId: string,
}