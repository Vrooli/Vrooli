import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const NoteModel = ({
    delegate: (prisma: PrismaType) => prisma.note,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'Note' as GraphQLModelType,
    validate: {} as any,
})