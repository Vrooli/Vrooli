import { bool, id, opt, req, YupModel, yupObj } from "../utils";
import { apiVersionValidation } from "./apiVersion";
import { labelValidation } from "./label";
import { tagValidation } from "./tag";

export const apiValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
    }, [
        ["ownedByUser", ["Connect"], "one", "opt"],
        ["ownedByOrganization", ["Connect"], "one", "opt"],
        ["parent", ["Connect"], "one", "opt"],
        ["tags", ["Connect", "Create"], "many", "opt", tagValidation],
        ["versions", ["Create"], "many", "opt", apiVersionValidation, ["root"]],
        ["labels", ["Connect", "Create"], "many", "opt", labelValidation],
    ], [["ownedByOrganizationConnect", "ownedByUserConnect"]], d),
    update: (d) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
    }, [
        ["ownedByUser", ["Connect"], "one", "opt"],
        ["ownedByOrganization", ["Connect"], "one", "opt"],
        ["tags", ["Connect", "Create", "Disconnect"], "many", "opt", tagValidation],
        ["versions", ["Create", "Update", "Delete"], "many", "opt", apiVersionValidation, ["root"]],
        ["labels", ["Connect", "Create", "Disconnect"], "many", "opt", labelValidation],
    ], [["ownedByOrganizationConnect", "ownedByUserConnect"]], d),
};
