import { Post, PostCreateInput, PostUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type PostShape = Pick<Post, "id"> & {
    __typename?: "Post";
}

export const shapePost: ShapeModel<PostShape, PostCreateInput, PostUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any,
};
