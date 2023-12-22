import { endpointPostPushDevice, PushDevice, PushDeviceCreateInput } from "@local/shared";
import { errorToMessage, fetchWrapper } from "api";
import { requestNotificationPermission, subscribeUserToPush } from "serviceWorkerRegistration";
import { getDeviceInfo } from "./display/device";
import { PubSub } from "./pubsub";

/**
 * Sets up push notifications for the user
 */
export const setupPush = async (showErrorWhenNotSupported = true) => {
    try {
        // Check if push notifications are supported (i.e. not in HTTP environment, and PWA is installed)
        if (window.location.protocol === "http:") {
            const message = "Notifications are not supported in http";
            if (showErrorWhenNotSupported) PubSub.get().publish("snack", { message, severity: "Error" });
            else console.warn(message);
            return;
        }
        if (!("serviceWorker" in navigator)) {
            const message = "Notifications are not supported in this browser";
            if (showErrorWhenNotSupported) PubSub.get().publish("snack", { message, severity: "Error" });
            else console.warn(message);
            return;
        }
        const result = await requestNotificationPermission();
        if (result === "denied") {
            PubSub.get().publish("snack", { messageKey: "PushPermissionDenied", severity: "Error" });
        }
        // Get subscription data
        const subscription: PushSubscription | null = await subscribeUserToPush();
        if (!subscription) {
            PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error" });
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
        fetchWrapper<PushDeviceCreateInput, PushDevice>({
            ...endpointPostPushDevice,
            inputs: {
                endpoint: subscription.endpoint,
                expires: subscription.expirationTime ?? undefined,
                keys: {
                    auth: authString,
                    p256dh: p256dhString,
                },
                name: getDeviceInfo().deviceName,
            },
            onError: (error) => { PubSub.get().publish("snack", { message: errorToMessage(error, ["en"]), severity: "Error", data: error }); },
        });
    } catch (error) {
        PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error", data: error });
    }
};
