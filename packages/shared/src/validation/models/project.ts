import { bool, handle, id, opt, permissions, req, YupModel, yupObj } from "../utils";
import { labelValidation } from "./label";
import { projectVersionValidation } from "./projectVersion";
import { tagValidation } from "./tag";

export const projectValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        handle: opt(handle),
        isPrivate: opt(bool),
        permissions: opt(permissions),
    }, [
        ["ownedByUser", ["Connect"], "one", "opt"],
        ["ownedByOrganization", ["Connect"], "one", "opt"],
        ["parent", ["Connect"], "one", "opt"],
        ["labels", ["Connect", "Create"], "many", "opt", labelValidation],
        ["versions", ["Create"], "many", "opt", projectVersionValidation, ["root"]],
        ["tags", ["Connect", "Create"], "many", "opt", tagValidation],
    ], [["ownedByOrganizationConnect", "ownedByUserConnect"]], o),
    update: ({ o }) => yupObj({
        id: req(id),
        handle: opt(handle),
        isPrivate: opt(bool),
        permissions: opt(permissions),
    }, [
        ["ownedByUser", ["Connect"], "one", "opt"],
        ["ownedByOrganization", ["Connect"], "one", "opt"],
        ["parent", ["Connect"], "one", "opt"],
        ["labels", ["Connect", "Create"], "many", "opt", labelValidation],
        ["versions", ["Create"], "many", "opt", projectVersionValidation, ["root"]],
        ["tags", ["Connect", "Create"], "many", "opt", tagValidation],
    ], [["ownedByOrganizationConnect", "ownedByUserConnect"]], o),
};
