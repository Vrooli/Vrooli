import { Prisma } from "@prisma/client";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { ModelLogic } from "./types";
import { UserModel } from "./user";

const __typename = 'StatsUser' as const;
const suppFields = [] as const;
export const StatsUserModel: ModelLogic<any, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_user,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            user: 'User',
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => UserModel.validate!.owner(data.user as any),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_userSelect>(data, [
            ['user', 'User'],
        ], languages),
        visibility: {
            private: { user: UserModel.validate!.visibility.private },
            public: { user: UserModel.validate!.visibility.public },
            owner: (userId) => ({ user: UserModel.validate!.visibility.owner(userId) }),
        }
    },
})