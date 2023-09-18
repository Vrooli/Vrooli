import { NotificationModelLogic } from "../base/types";
import { Formatter } from "../types";

export const NotificationFormat: Formatter<NotificationModelLogic> = {
    gqlRelMap: {
        __typename: "Notification",
    },
    prismaRelMap: {
        __typename: "Notification",
    },
    countFields: {},
};
