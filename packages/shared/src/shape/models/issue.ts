import { Issue, IssueCreateInput, IssueFor, IssueTranslation, IssueTranslationCreateInput, IssueTranslationUpdateInput, IssueUpdateInput } from "../../api/generated/graphqlTypes";
import { CanConnect, ShapeModel } from "../../consts/commonTypes";
import { LabelShape, shapeLabel } from "./label";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type IssueTranslationShape = Pick<IssueTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "IssueTranslation";
}

export type IssueShape = Pick<Issue, "id"> & {
    __typename: "Issue";
    issueFor: IssueFor;
    for: { id: string };
    labels?: CanConnect<LabelShape>[] | null;
    translations: IssueTranslationShape[];
}

export const shapeIssueTranslation: ShapeModel<IssueTranslationShape, IssueTranslationCreateInput, IssueTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "name")),
};

export const shapeIssue: ShapeModel<IssueShape, IssueCreateInput, IssueUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "issueFor"),
        ...createRel(d, "for", ["Connect"], "one"),
        ...createRel(d, "labels", ["Connect", "Create"], "many", shapeLabel),
        ...createRel(d, "translations", ["Create"], "many", shapeIssueTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateRel(o, u, "labels", ["Connect", "Disconnect", "Create"], "many", shapeLabel),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeIssueTranslation),
    }),
};
