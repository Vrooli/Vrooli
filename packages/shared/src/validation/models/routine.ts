import { bool, id, opt, permissions, req, YupModel, yupObj } from "../utils";
import { labelValidation } from "./label";
import { routineVersionValidation } from "./routineVersion";
import { tagValidation } from "./tag";

export const routineValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        isInternal: opt(bool),
        isPrivate: opt(bool),
        permissions: req(permissions),
    }, [
        ["user", ["Connect"], "one", "opt"],
        ["organization", ["Connect"], "one", "opt"],
        ["parent", ["Connect"], "one", "opt"],
        ["labels", ["Connect", "Create"], "many", "opt", labelValidation],
        ["tags", ["Connect", "Create"], "many", "opt", tagValidation],
        ["versions", ["Create"], "many", "req", routineVersionValidation, ["root"]],
    ], [["organizationConnect", "userConnect"]], o),
    update: ({ o }) => yupObj({
        id: req(id),
        isInternal: opt(bool),
        isPrivate: opt(bool),
        permissions: opt(permissions),
    }, [
        ["user", ["Connect"], "one", "opt"],
        ["organization", ["Connect"], "one", "opt"],
        ["labels", ["Connect", "Create", "Disconnect"], "many", "opt", labelValidation],
        ["tags", ["Connect", "Create", "Disconnect"], "many", "opt", tagValidation],
        ["versions", ["Create", "Update", "Delete"], "many", "req", routineVersionValidation, ["root"]],
    ], [["organizationConnect", "userConnect"]], o),
};
