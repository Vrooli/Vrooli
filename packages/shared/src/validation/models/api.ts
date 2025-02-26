import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, id } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { apiVersionValidation } from "./apiVersion.js";
import { labelValidation } from "./label.js";
import { tagValidation } from "./tag.js";

export const apiValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
    }, [
        ["ownedByUser", ["Connect"], "one", "opt"],
        ["ownedByTeam", ["Connect"], "one", "opt"],
        ["parent", ["Connect"], "one", "opt"],
        ["tags", ["Connect", "Create"], "many", "opt", tagValidation],
        ["versions", ["Create"], "many", "opt", apiVersionValidation, ["root"]],
        ["labels", ["Connect", "Create"], "many", "opt", labelValidation],
    ], [["ownedByTeamConnect", "ownedByUserConnect", true]], d),
    update: (d) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
    }, [
        ["ownedByUser", ["Connect"], "one", "opt"],
        ["ownedByTeam", ["Connect"], "one", "opt"],
        ["tags", ["Connect", "Create", "Disconnect"], "many", "opt", tagValidation],
        ["versions", ["Create", "Update", "Delete"], "many", "opt", apiVersionValidation, ["root"]],
        ["labels", ["Connect", "Create", "Disconnect"], "many", "opt", labelValidation],
    ], [["ownedByTeamConnect", "ownedByUserConnect", false]], d),
};
