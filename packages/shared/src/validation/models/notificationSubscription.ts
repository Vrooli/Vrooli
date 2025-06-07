/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in notificationSubscription.test.ts
import { SubscribableObject } from "../../api/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, id } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

function subscribableObject() {
    return enumToYup(SubscribableObject);
}

export const notificationSubscriptionValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        silent: opt(bool),
        objectType: req(subscribableObject()),
    }, [
        ["object", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        silent: opt(bool),
    }, [], [], d),
};
/* c8 ignore stop */
