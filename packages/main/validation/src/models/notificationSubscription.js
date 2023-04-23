import { SubscribableObject } from "@local/consts";
import { bool, enumToYup, id, opt, req, yupObj } from "../utils";
const subscribableObject = enumToYup(SubscribableObject);
export const notificationSubscriptionValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        silent: opt(bool),
        objectType: req(subscribableObject),
    }, [
        ["object", ["Connect"], "one", "req"],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        silent: opt(bool),
    }, [], [], o),
};
//# sourceMappingURL=notificationSubscription.js.map