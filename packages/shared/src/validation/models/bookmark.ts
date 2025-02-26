import { BookmarkFor } from "../../api/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { bookmarkListValidation } from "./bookmarkList.js";

const bookmarkFor = enumToYup(BookmarkFor);

export const bookmarkValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        bookmarkFor: req(bookmarkFor),
    }, [
        ["for", ["Connect"], "one", "req"],
        ["list", ["Connect", "Create"], "one", "opt", bookmarkListValidation, ["bookmarks"]],
    ], [["listConnect", "listCreate", true]], d),
    update: (d) => yupObj({
        id: req(id),
    }, [
        ["list", ["Connect", "Update"], "one", "opt", bookmarkListValidation, ["bookmarks"]],
    ], [], d),
};
