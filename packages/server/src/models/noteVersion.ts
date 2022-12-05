import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const NoteVersionModel = ({
    delegate: (prisma: PrismaType) => prisma.note_version,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'NoteVersion' as GraphQLModelType,
    validate: {} as any,
})