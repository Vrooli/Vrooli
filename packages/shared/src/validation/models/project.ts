import { bool, handle, id, opt, permissions, req, YupModel, yupObj } from "../utils";
import { labelValidation } from "./label";
import { projectVersionValidation } from "./projectVersion";
import { tagValidation } from "./tag";

export const projectValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        handle: opt(handle),
        isPrivate: opt(bool),
        permissions: opt(permissions),
    }, [
        ["ownedByUser", ["Connect"], "one", "opt"],
        ["ownedByTeam", ["Connect"], "one", "opt"],
        ["parent", ["Connect"], "one", "opt"],
        ["labels", ["Connect", "Create"], "many", "opt", labelValidation],
        ["versions", ["Create"], "many", "opt", projectVersionValidation, ["root"]],
        ["tags", ["Connect", "Create"], "many", "opt", tagValidation],
    ], [["ownedByTeamConnect", "ownedByUserConnect", true]], d),
    update: (d) => yupObj({
        id: req(id),
        handle: opt(handle),
        isPrivate: opt(bool),
        permissions: opt(permissions),
    }, [
        ["ownedByUser", ["Connect"], "one", "opt"],
        ["ownedByTeam", ["Connect"], "one", "opt"],
        ["parent", ["Connect"], "one", "opt"],
        ["labels", ["Connect", "Create"], "many", "opt", labelValidation],
        ["versions", ["Create"], "many", "opt", projectVersionValidation, ["root"]],
        ["tags", ["Connect", "Create"], "many", "opt", tagValidation],
    ], [["ownedByTeamConnect", "ownedByUserConnect", false]], d),
};
