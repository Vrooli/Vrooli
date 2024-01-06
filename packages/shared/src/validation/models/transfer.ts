import { TransferObjectType, YupMutateParams } from "@local/shared";
import { YupModel, enumToYup, id, message, opt, req, yupObj } from "../utils";

const transferObjectType = enumToYup(TransferObjectType);

export const transferRequestSendValidation = (d: YupMutateParams) => yupObj({
    id: req(id),
    message: opt(message),
    objectType: req(transferObjectType),
}, [
    ["object", ["Connect"], "one", "req"],
    ["toOrganization", ["Connect"], "one", "opt"],
    ["toUser", ["Connect"], "one", "opt"],
], [["toOrganizationConnect", "toUserConnect"]], d);

export const transferRequestReceiveValidation = (d: YupMutateParams) => yupObj({
    id: req(id),
    message: opt(message),
    objectType: req(transferObjectType),
}, [
    ["object", ["Connect"], "one", "req"],
    ["toOrganization", ["Connect"], "one", "opt"],
], [], d);

export const transferValidation: YupModel<["update"]> = {
    // Cannot create a transfer through normal means. Must use request send/receive
    update: (d) => yupObj({
        id: req(id),
        message: opt(message),
    }, [], [], d),
};
