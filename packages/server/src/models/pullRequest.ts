import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const PullRequestModel = ({
    delegate: (prisma: PrismaType) => prisma.pull_request,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'PullRequest' as GraphQLModelType,
    validate: {} as any,
})