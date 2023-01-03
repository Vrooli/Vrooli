import { Issue, IssueCreateInput, IssueUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type IssueShape = Pick<Issue, 'id'>

export const shapeIssue: ShapeModel<IssueShape, IssueCreateInput, IssueUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}