import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const IssueModel = ({
    delegate: (prisma: PrismaType) => prisma.issue,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'Issue' as GraphQLModelType,
    validate: {} as any,
})