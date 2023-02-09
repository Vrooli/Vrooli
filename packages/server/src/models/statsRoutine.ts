import { Prisma } from "@prisma/client";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { RoutineModel } from "./routine";
import { ModelLogic } from "./types";

const __typename = 'StatsRoutine' as const;
const suppFields = [] as const;
export const StatsRoutineModel: ModelLogic<any, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_routine,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            routine: 'Routine',
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => RoutineModel.validate!.owner(data.routine as any),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_routineSelect>(data, [
            ['routine', 'Routine'],
        ], languages),
        visibility: {
            private: { routine: RoutineModel.validate!.visibility.private },
            public: { routine: RoutineModel.validate!.visibility.public },
            owner: (userId) => ({ routine: RoutineModel.validate!.visibility.owner(userId) }),
        }
    },
})