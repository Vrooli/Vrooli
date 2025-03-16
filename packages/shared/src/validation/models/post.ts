import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, id } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { resourceListValidation } from "./resourceList.js";
import { tagValidation } from "./tag.js";

export const postValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        isPinned: opt(bool),
        isPrivate: opt(bool),
    }, [
        ["team", ["Connect"], "one", "opt"],
        ["user", ["Connect"], "one", "opt"],
        ["repostedFrom", ["Connect"], "one", "opt"],
        ["resourceList", ["Create"], "one", "opt", resourceListValidation],
        ["tags", ["Connect", "Create"], "many", "opt", tagValidation],
    ], [["teamConnect", "userConnect", true]], d),
    update: (d) => yupObj({
        id: req(id),
        isPinned: opt(bool),
        isPrivate: opt(bool),
    }, [
        ["resourceList", ["Update"], "one", "opt", resourceListValidation],
        ["tags", ["Connect", "Create", "Disconnect"], "many", "opt", tagValidation],
    ], [], d),
};
