import { pushDeviceCreate } from "../api/generated/endpoints/pushDevice_create";
import { documentNodeWrapper, errorToCode } from "../api/utils";
import { requestNotificationPermission, subscribeUserToPush } from "../serviceWorkerRegistration";
import { getDeviceInfo } from "./display/device";
import { PubSub } from "./pubsub";
export const setupPush = async () => {
    const result = await requestNotificationPermission();
    if (result === "denied") {
        PubSub.get().publishSnack({ messageKey: "PushPermissionDenied", severity: "Error" });
    }
    const subscription = await subscribeUserToPush();
    if (!subscription) {
        PubSub.get().publishSnack({ messageKey: "ErrorUnknown", severity: "Error" });
        return;
    }
    const p256dhArray = Array.from(new Uint8Array(subscription.getKey("p256dh") ?? new ArrayBuffer(0)));
    const p256dhString = p256dhArray.map((b) => String.fromCharCode(b)).join("");
    const authArray = Array.from(new Uint8Array(subscription.getKey("auth") ?? new ArrayBuffer(0)));
    const authString = authArray.map((b) => String.fromCharCode(b)).join("");
    console.log("got subscription", subscription, subscription.getKey("p256dh")?.toString(), subscription.getKey("auth")?.toString());
    documentNodeWrapper({
        node: pushDeviceCreate,
        input: {
            endpoint: subscription.endpoint,
            expires: subscription.expirationTime ?? undefined,
            keys: {
                auth: authString,
                p256dh: p256dhString,
            },
            name: getDeviceInfo().deviceName,
        },
        successMessage: () => ({ key: "PushDeviceCreated" }),
        onError: (error) => { PubSub.get().publishSnack({ messageKey: errorToCode(error), severity: "Error", data: error }); },
    });
};
//# sourceMappingURL=push.js.map