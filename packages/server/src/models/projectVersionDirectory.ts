import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const ProjectVersionDirectoryModel = ({
    delegate: (prisma: PrismaType) => prisma.project_version_directory,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'ProjectVersionDirectory' as GraphQLModelType,
    validate: {} as any,
})