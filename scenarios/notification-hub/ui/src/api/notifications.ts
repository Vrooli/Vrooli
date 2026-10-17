import { createClient, type Client } from "@connectrpc/connect";
import { createScenarioConnectTransport } from "@vrooli/api-base";
import { ConversationsService } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { DeliveryService } from "@vrooli/proto-types/notification-hub/v1/delivery/delivery_pb";
import { IdentityService } from "@vrooli/proto-types/notification-hub/v1/identity/identity_pb";
import { NotificationsService } from "@vrooli/proto-types/notification-hub/v1/notifications/notifications_pb";
import { RecipientsService } from "@vrooli/proto-types/notification-hub/v1/recipients/recipients_pb";

/**
 * The owner session is an HttpOnly same-origin cookie set by the hub's
 * IdentityService; the browser sends it on every same-origin call and page
 * scripts never see the token. The header tells sign-in not to echo tokens
 * into a response body.
 */
const sessionFetch: typeof fetch = (input, init) => {
  const headers = new Headers(init?.headers);
  headers.set("X-Vrooli-Browser-Session", "1");
  return fetch(input, { ...init, headers, credentials: "same-origin" });
};

const transport = createScenarioConnectTransport({ fetch: sessionFetch });
export const notificationsClient: Client<typeof NotificationsService> = createClient(NotificationsService, transport);
export const deliveryClient: Client<typeof DeliveryService> = createClient(DeliveryService, transport);
export const recipientsClient: Client<typeof RecipientsService> = createClient(RecipientsService, transport);
export const conversationsClient: Client<typeof ConversationsService> = createClient(ConversationsService, transport);
export const identityClient: Client<typeof IdentityService> = createClient(IdentityService, transport);

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
