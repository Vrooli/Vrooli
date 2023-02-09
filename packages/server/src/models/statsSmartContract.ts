import { Prisma } from "@prisma/client";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { SmartContractModel } from "./smartContract";
import { ModelLogic } from "./types";

const __typename = 'StatsSmartContract' as const;
const suppFields = [] as const;
export const StatsSmartContractModel: ModelLogic<any, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_smart_contract,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            smartContract: 'SmartContract',
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => SmartContractModel.validate!.owner(data.smartContract as any),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_smart_contractSelect>(data, [
            ['smartContract', 'SmartContract'],
        ], languages),
        visibility: {
            private: { smartContract: SmartContractModel.validate!.visibility.private },
            public: { smartContract: SmartContractModel.validate!.visibility.public },
            owner: (userId) => ({ smartContract: SmartContractModel.validate!.visibility.owner(userId) }),
        }
    },
})