import { Prisma } from "@prisma/client";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { StandardModel } from "./standard";
import { ModelLogic } from "./types";

const __typename = 'StatsStandard' as const;
const suppFields = [] as const;
export const StatsStandardModel: ModelLogic<any, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_standard,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            standard: 'Standard',
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => StandardModel.validate!.owner(data.standard as any),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_standardSelect>(data, [
            ['standard', 'Standard'],
        ], languages),
        visibility: {
            private: { standard: StandardModel.validate!.visibility.private },
            public: { standard: StandardModel.validate!.visibility.public },
            owner: (userId) => ({ standard: StandardModel.validate!.visibility.owner(userId) }),
        }
    },
})