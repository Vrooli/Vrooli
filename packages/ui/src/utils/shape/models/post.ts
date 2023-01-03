import { Post, PostCreateInput, PostUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type PostShape = Pick<Post, 'id'>

export const shapePost: ShapeModel<PostShape, PostCreateInput, PostUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}