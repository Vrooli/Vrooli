import { Post, PostCreateInput, PostUpdateInput } from "../../api/generated/graphqlTypes";
import { ShapeModel } from "../../consts/commonTypes";
import { shapeUpdate } from "./tools";

export type PostShape = Pick<Post, "id"> & {
    __typename: "Post";
}

export const shapePost: ShapeModel<PostShape, PostCreateInput, PostUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any,
};
