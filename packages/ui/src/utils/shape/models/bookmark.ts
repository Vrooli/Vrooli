import { Bookmark, BookmarkCreateInput, BookmarkFor, BookmarkUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { BookmarkListShape, shapeBookmarkList } from "./bookmarkList";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type BookmarkShape = Pick<Bookmark, "id"> & {
    __typename?: "Bookmark";
    to: { __typename: BookmarkFor, id: string };
    list: { id: string } | BookmarkListShape;
}

export const shapeBookmark: ShapeModel<BookmarkShape, BookmarkCreateInput, BookmarkUpdateInput> = {
    create: (d) => ({
        forConnect: d.to.id,
        bookmarkFor: d.to.__typename,
        ...createPrims(d, "id"),
        ...createRel(d, "list", ["Create", "Connect"], "one", shapeBookmarkList),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateRel(o, u, "list", ["Create", "Update"], "one", shapeBookmarkList),
    }, a),
};
