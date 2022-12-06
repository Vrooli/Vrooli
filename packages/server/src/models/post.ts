import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const PostModel = ({
    delegate: (prisma: PrismaType) => prisma.post,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'Post' as GraphQLModelType,
    validate: {} as any,
})