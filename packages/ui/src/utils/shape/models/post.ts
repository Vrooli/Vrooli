import { Post, PostCreateInput, PostUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type PostShape = Pick<Post, 'id'>

export const shapePost: ShapeModel<PostShape, PostCreateInput, PostUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}