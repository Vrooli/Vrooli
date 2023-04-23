import { TransferObjectType } from "@local/consts";
import { enumToYup, id, message, opt, req, yupObj } from "../utils";
const transferObjectType = enumToYup(TransferObjectType);
export const transferRequestSendValidation = ({ o }) => yupObj({
    id: req(id),
    message: opt(message),
    objectType: req(transferObjectType),
}, [
    ["object", ["Connect"], "one", "req"],
    ["toOrganization", ["Connect"], "one", "opt"],
    ["toUser", ["Connect"], "one", "opt"],
], [["toOrganizationConnect", "toUserConnect"]], o);
export const transferRequestReceiveValidation = ({ o }) => yupObj({
    id: req(id),
    message: opt(message),
    objectType: req(transferObjectType),
}, [
    ["object", ["Connect"], "one", "req"],
    ["toOrganization", ["Connect"], "one", "opt"],
], [], o);
export const transferValidation = {
    update: ({ o }) => yupObj({
        id: req(id),
        message: opt(message),
    }, [], [], o),
};
//# sourceMappingURL=transfer.js.map