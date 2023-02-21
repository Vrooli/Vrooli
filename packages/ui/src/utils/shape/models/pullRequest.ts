import { PullRequest, PullRequestCreateInput, PullRequestTranslation, PullRequestTranslationCreateInput, PullRequestTranslationUpdateInput, PullRequestUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, shapeUpdate, updatePrims } from "./tools";

export type PullRequestTranslationShape = Pick<PullRequestTranslation, 'id' | 'language' | 'text'>

export type PullRequestShape = Pick<PullRequest, 'id'>

export const shapePullRequestTranslation: ShapeModel<PullRequestTranslationShape, PullRequestTranslationCreateInput, PullRequestTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'text'),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, 'id', 'text'), a)
}

export const shapePullRequest: ShapeModel<PullRequestShape, PullRequestCreateInput, PullRequestUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}