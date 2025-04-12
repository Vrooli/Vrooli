import { endpointsPushDevice, PushDevice, PushDeviceCreateInput } from "@local/shared";
import { fetchData } from "../api/fetchData.js";
import { ServerResponseParser } from "../api/responseParser.js";
import { requestNotificationPermission, subscribeUserToPush } from "../serviceWorkerRegistration.js";
import { getDeviceInfo } from "./display/device.js";
import { PubSub } from "./pubsub.js";

/**
 * Sets up push notifications for the user
 */
export async function setupPush(showErrorWhenNotSupported = true): Promise<PushDevice | undefined> {
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
        try {
            const response = await fetchData<PushDeviceCreateInput, PushDevice>({
                ...endpointsPushDevice.createOne,
                inputs: {
                    endpoint: subscription.endpoint,
                    expires: subscription.expirationTime ?? undefined,
                    keys: {
                        auth: authString,
                        p256dh: p256dhString,
                    },
                    name: getDeviceInfo().deviceName,
                },
            });
            if (response.data) {
                console.log("got response", response.data);
                PubSub.get().publish("snack", { messageKey: "PushDeviceCreated", severity: "Success" });
                return response.data;
            } else if (response.errors) {
                ServerResponseParser.displayErrors(response.errors);
            } else {
                PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error" });
            }
        } catch (error) {
            PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error", data: error });
        }
    } catch (error) {
        PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error", data: error });
    }
}
