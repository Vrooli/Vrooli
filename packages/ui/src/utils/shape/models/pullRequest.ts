import { PullRequest, PullRequestCreateInput, PullRequestUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type PullRequestShape = Pick<PullRequest, 'id'>

export const shapePullRequest: ShapeModel<PullRequestShape, PullRequestCreateInput, PullRequestUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any
}