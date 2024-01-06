import { BookmarkFor } from "@local/shared";
import { enumToYup, id, req, YupModel, yupObj } from "../utils";
import { bookmarkListValidation } from "./bookmarkList";

const bookmarkFor = enumToYup(BookmarkFor);

export const bookmarkValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        bookmarkFor: req(bookmarkFor),
    }, [
        ["for", ["Connect"], "one", "req"],
        ["list", ["Connect", "Create"], "one", "opt", bookmarkListValidation, ["bookmarks"]],
    ], [["listConnect", "listCreate"]], d),
    update: (d) => yupObj({
        id: req(id),
    }, [
        ["list", ["Connect", "Update"], "one", "opt", bookmarkListValidation, ["bookmarks"]],
    ], [], d),
};
