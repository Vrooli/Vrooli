import { Prisma } from "@prisma/client";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { ApiModel } from "./api";
import { ModelLogic } from "./types";

const __typename = 'StatsApi' as const;
const suppFields = [] as const;
export const StatsApiModel: ModelLogic<any, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_api,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            api: 'Api',
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ApiModel.validate!.owner(data.api as any),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_apiSelect>(data, [
            ['api', 'Api'],
        ], languages),
        visibility: {
            private: { api: ApiModel.validate!.visibility.private },
            public: { api: ApiModel.validate!.visibility.public },
            owner: (userId) => ({ api: ApiModel.validate!.visibility.owner(userId) }),
        }
    },
})