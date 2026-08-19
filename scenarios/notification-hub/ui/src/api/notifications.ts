import { createClient, type Client } from "@connectrpc/connect";
import { createScenarioConnectTransport } from "@vrooli/api-base";
import { DeliveryService } from "@vrooli/proto-types/notification-hub/v1/delivery/delivery_pb";
import { NotificationsService } from "@vrooli/proto-types/notification-hub/v1/notifications/notifications_pb";
import { RecipientsService } from "@vrooli/proto-types/notification-hub/v1/recipients/recipients_pb";

const identityFetch: typeof fetch = (input, init) => {
  const headers = new Headers(init?.headers);
  headers.set("X-Vrooli-Identity-Subject", localStorage.getItem("notification-hub.identity") || "owner");
  return fetch(input, { ...init, headers });
};

const transport = createScenarioConnectTransport({ fetch: identityFetch });
export const notificationsClient: Client<typeof NotificationsService> = createClient(NotificationsService, transport);
export const deliveryClient: Client<typeof DeliveryService> = createClient(DeliveryService, transport);
export const recipientsClient: Client<typeof RecipientsService> = createClient(RecipientsService, transport);

export async function registerBrowserPushSubscription(applicationServerKey: BufferSource): Promise<PushSubscriptionJSON> {
  const registration = await navigator.serviceWorker.ready;
  const subscription = await registration.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey });
  const json = subscription.toJSON();
  if (!json.endpoint || !json.keys?.p256dh || !json.keys.auth) {
    throw new Error("browser returned an incomplete push subscription");
  }
  await recipientsClient.registerPushSubscription({
    endpoint: json.endpoint,
    p256dh: json.keys.p256dh,
    auth: json.keys.auth,
    origin: window.location.origin,
  });
  return json;
}
