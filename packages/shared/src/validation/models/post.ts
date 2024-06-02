import { bool, id, opt, req, YupModel, yupObj } from "../utils";
import { resourceListValidation } from "./resourceList";
import { tagValidation } from "./tag";

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
