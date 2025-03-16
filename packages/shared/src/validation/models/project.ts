import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, handle, id, permissions } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { labelValidation } from "./label.js";
import { projectVersionValidation } from "./projectVersion.js";
import { tagValidation } from "./tag.js";

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
