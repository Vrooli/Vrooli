import { id, req, opt, YupModel, permissions, yupObj, bool } from "../utils";
import { labelValidation } from "./label";
import { tagValidation } from "./tag";
import { smartContractVersionValidation } from "./smartContractVersion";

export const smartContractValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
        permissions: req(permissions),
    }, [
        ["user", ["Connect"], "one", "opt"],
        ["organization", ["Connect"], "one", "opt"],
        ["parent", ["Connect"], "one", "opt"],
        ["labels", ["Connect", "Create"], "many", "opt", labelValidation],
        ["tags", ["Connect", "Create"], "many", "opt", tagValidation],
        ["versions", ["Create"], "many", "req", smartContractVersionValidation, ["root"]],
    ], [["organizationConnect", "userConnect"]], o),
    update: ({ o }) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
        permissions: opt(permissions),
    }, [
        ["user", ["Connect"], "one", "opt"],
        ["organization", ["Connect"], "one", "opt"],
        ["labels", ["Connect", "Create", "Disconnect"], "many", "opt", labelValidation],
        ["tags", ["Connect", "Create", "Disconnect"], "many", "opt", tagValidation],
        ["versions", ["Create", "Update", "Delete"], "many", "req", smartContractVersionValidation, ["root"]],
    ], [["organizationConnect", "userConnect"]], o),
};