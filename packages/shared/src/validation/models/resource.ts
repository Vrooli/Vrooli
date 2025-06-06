/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in resource.test.ts
import { ResourceType } from "../../api/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, id, permissions, publicId } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { resourceVersionValidation } from "./resourceVersion.js";
import { tagValidation } from "./tag.js";

const resourceType = enumToYup(ResourceType);

export const resourceValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        publicId: opt(publicId),
        isInternal: opt(bool),
        isPrivate: opt(bool),
        permissions: opt(permissions),
        resourceType: req(resourceType),
    }, [
        ["ownedByUser", ["Connect"], "one", "opt"],
        ["ownedByTeam", ["Connect"], "one", "opt"],
        ["parent", ["Connect"], "one", "opt"],
        ["tags", ["Connect", "Create"], "many", "opt", tagValidation],
        ["versions", ["Create"], "many", "req", resourceVersionValidation, ["root"]],
    ], [["ownedByTeamConnect", "ownedByUserConnect", true]], d),
    update: (d) => yupObj({
        id: req(id),
        isInternal: opt(bool),
        isPrivate: opt(bool),
        permissions: opt(permissions),
    }, [
        ["ownedByUser", ["Connect"], "one", "opt"],
        ["ownedByTeam", ["Connect"], "one", "opt"],
        ["tags", ["Connect", "Create", "Disconnect"], "many", "opt", tagValidation],
        ["versions", ["Create", "Update", "Delete"], "many", "opt", resourceVersionValidation, ["root"]],
    ], [["ownedByTeamConnect", "ownedByUserConnect", false]], d),
};
/* c8 ignore stop */
