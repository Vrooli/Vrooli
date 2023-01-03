import { PullRequest, PullRequestCreateInput, PullRequestUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type PullRequestShape = Pick<PullRequest, 'id'>

export const shapePullRequest: ShapeModel<PullRequestShape, PullRequestCreateInput, PullRequestUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}