import { TransferObjectType } from "../../api/types";
import { YupMutateParams } from "../../validation/utils/types";
import { YupModel, enumToYup, id, message, opt, req, yupObj } from "../utils";

const transferObjectType = enumToYup(TransferObjectType);

export const transferRequestSendValidation = (d: YupMutateParams) => yupObj({
    id: req(id),
    message: opt(message),
    objectType: req(transferObjectType),
}, [
    ["object", ["Connect"], "one", "req"],
    ["toTeam", ["Connect"], "one", "opt"],
    ["toUser", ["Connect"], "one", "opt"],
], [["toTeamConnect", "toUserConnect", true]], d);

export const transferRequestReceiveValidation = (d: YupMutateParams) => yupObj({
    id: req(id),
    message: opt(message),
    objectType: req(transferObjectType),
}, [
    ["object", ["Connect"], "one", "req"],
    ["toTeam", ["Connect"], "one", "opt"],
], [], d);

export const transferValidation: YupModel<["update"]> = {
    // Cannot create a transfer through normal means. Must use request send/receive
    update: (d) => yupObj({
        id: req(id),
        message: opt(message),
    }, [], [], d),
};
