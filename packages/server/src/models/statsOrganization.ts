import { Prisma } from "@prisma/client";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { OrganizationModel } from "./organization";
import { ModelLogic } from "./types";

const __typename = 'StatsOrganization' as const;
const suppFields = [] as const;
export const StatsOrganizationModel: ModelLogic<any, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_organization,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            organization: 'Organization',
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => OrganizationModel.validate!.owner(data.organization as any),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_organizationSelect>(data, [
            ['organization', 'Organization'],
        ], languages),
        visibility: {
            private: { organization: OrganizationModel.validate!.visibility.private },
            public: { organization: OrganizationModel.validate!.visibility.public },
            owner: (userId) => ({ organization: OrganizationModel.validate!.visibility.owner(userId) }),
        }
    },
})