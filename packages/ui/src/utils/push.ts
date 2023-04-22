import { PushDevice, PushDeviceCreateInput } from "@shared/consts";
import { requestNotificationPermission, subscribeUserToPush } from "serviceWorkerRegistration";
import { pushDeviceCreate } from "../api/generated/endpoints/pushDevice_create";
import { documentNodeWrapper, errorToCode } from "../api/utils";
import { getDeviceInfo } from "./display/device";
import { PubSub } from "./pubsub";

/**
 * Sets up push notifications for the user
 */
export const setupPush = async () => {
    const result = await requestNotificationPermission();
    if (result === "denied") {
        PubSub.get().publishSnack({ messageKey: "PushPermissionDenied", severity: "Error" });
    }
    // Get subscription data
    const subscription: PushSubscription | null = await subscribeUserToPush();
    if (!subscription) {
        PubSub.get().publishSnack({ messageKey: "ErrorUnknown", severity: "Error" });
        return;
    }
    // Converting p256dh to a string
    const p256dhArray = Array.from(new Uint8Array(subscription.getKey("p256dh") ?? new ArrayBuffer(0)));
    const p256dhString = p256dhArray.map((b) => String.fromCharCode(b)).join("");
    // Converting auth to a string
    const authArray = Array.from(new Uint8Array(subscription.getKey("auth") ?? new ArrayBuffer(0)));
    const authString = authArray.map((b) => String.fromCharCode(b)).join("");
    // Call pushDeviceCreate
    console.log("got subscription", subscription, subscription.getKey("p256dh")?.toString(), subscription.getKey("auth")?.toString());
    documentNodeWrapper<PushDevice, PushDeviceCreateInput>({
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
        onError: (error) => { PubSub.get().publishSnack({ messageKey: errorToCode(error), severity: "Error", data: error }); }
    })
};