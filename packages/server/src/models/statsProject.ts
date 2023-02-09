import { Prisma } from "@prisma/client";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { ProjectModel } from "./project";
import { ModelLogic } from "./types";

const __typename = 'StatsProject' as const;
const suppFields = [] as const;
export const StatsProjectModel: ModelLogic<any, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_project,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            project: 'Project',
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ProjectModel.validate!.owner(data.project as any),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_projectSelect>(data, [
            ['project', 'Project'],
        ], languages),
        visibility: {
            private: { project: ProjectModel.validate!.visibility.private },
            public: { project: ProjectModel.validate!.visibility.public },
            owner: (userId) => ({ project: ProjectModel.validate!.visibility.owner(userId) }),
        }
    },
})