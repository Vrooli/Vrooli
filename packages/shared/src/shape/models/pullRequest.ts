import { PullRequest, PullRequestCreateInput, PullRequestTranslation, PullRequestTranslationCreateInput, PullRequestTranslationUpdateInput, PullRequestUpdateInput } from "../../api/generated/graphqlTypes";
import { ShapeModel } from "../../consts/commonTypes";
import { createPrims, shapeUpdate, updateTranslationPrims } from "./tools";

export type PullRequestTranslationShape = Pick<PullRequestTranslation, "id" | "language" | "text"> & {
    __typename?: "PullRequestTranslation";
}

export type PullRequestShape = Pick<PullRequest, "id"> & {
    __typename: "PullRequest";
}

export const shapePullRequestTranslation: ShapeModel<PullRequestTranslationShape, PullRequestTranslationCreateInput, PullRequestTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "text"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "text")),
};

export const shapePullRequest: ShapeModel<PullRequestShape, PullRequestCreateInput, PullRequestUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any,
};
