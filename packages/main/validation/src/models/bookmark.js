import { BookmarkFor } from "@local/consts";
import { enumToYup, id, req, yupObj } from "../utils";
import { bookmarkListValidation } from "./bookmarkList";
const bookmarkFor = enumToYup(BookmarkFor);
export const bookmarkValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        bookmarkFor: req(bookmarkFor),
    }, [
        ["for", ["Connect"], "one", "req"],
        ["list", ["Connect", "Create"], "one", "opt", bookmarkListValidation, ["bookmarks"]],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
    }, [
        ["list", ["Connect", "Update"], "one", "opt", bookmarkListValidation, ["bookmarks"]],
    ], [], o),
};
//# sourceMappingURL=bookmark.js.map