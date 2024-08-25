import { bool, id, opt, permissions, req, YupModel, yupObj } from "../utils";
import { codeVersionValidation } from "./codeVersion";
import { labelValidation } from "./label";
import { tagValidation } from "./tag";

export const codeValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
        permissions: opt(permissions),
    }, [
        ["ownedByUser", ["Connect"], "one", "opt"],
        ["ownedByTeam", ["Connect"], "one", "opt"],
        ["parent", ["Connect"], "one", "opt"],
        ["labels", ["Connect", "Create"], "many", "opt", labelValidation],
        ["tags", ["Connect", "Create"], "many", "opt", tagValidation],
        ["versions", ["Create"], "many", "req", codeVersionValidation, ["root"]],
    ], [["ownedByTeamConnect", "ownedByUserConnect", true]], d),
    update: (d) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
        permissions: opt(permissions),
    }, [
        ["ownedByUser", ["Connect"], "one", "opt"],
        ["ownedByTeam", ["Connect"], "one", "opt"],
        ["labels", ["Connect", "Create", "Disconnect"], "many", "opt", labelValidation],
        ["tags", ["Connect", "Create", "Disconnect"], "many", "opt", tagValidation],
        ["versions", ["Create", "Update", "Delete"], "many", "req", codeVersionValidation, ["root"]],
    ], [["ownedByTeamConnect", "ownedByUserConnect", false]], d),
};
