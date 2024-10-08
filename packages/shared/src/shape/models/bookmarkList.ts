import { BookmarkList, BookmarkListCreateInput, BookmarkListUpdateInput } from "../../api/generated/graphqlTypes";
import { ShapeModel } from "../../consts/commonTypes";
import { BookmarkShape, shapeBookmark } from "./bookmark";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type BookmarkListShape = Pick<BookmarkList, "id" | "label"> & {
    __typename: "BookmarkList";
    bookmarks?: BookmarkShape[] | null;
}

export const shapeBookmarkList: ShapeModel<BookmarkListShape, BookmarkListCreateInput, BookmarkListUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "label"),
        ...createRel(d, "bookmarks", ["Create"], "many", shapeBookmark),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "label"),
        ...updateRel(o, u, "bookmarks", ["Create", "Delete"], "many", shapeBookmark),
    }),
};
