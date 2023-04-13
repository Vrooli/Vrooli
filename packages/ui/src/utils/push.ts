import { PushDevice, PushDeviceCreateInput } from '@shared/consts';
import { pushDeviceCreate } from 'api/generated/endpoints/pushDevice_create';
import { documentNodeWrapper, errorToCode } from "api/utils";
import { requestNotificationPermission, subscribeUserToPush } from 'serviceWorkerRegistration';
import { PubSub } from './pubsub';

/**
 * Sets up push notifications for the user
 */
export const setupPush = async () => {
    const result = await requestNotificationPermission();
    if (result === 'denied') {
        PubSub.get().publishSnack({ messageKey: 'PushPermissionDenied', severity: 'Error' });
    }
    // Get subscription data
    const subscription: PushSubscription | null = await subscribeUserToPush();
    if (!subscription) {
        PubSub.get().publishSnack({ messageKey: 'ErrorUnknown', severity: 'Error' });
        return;
    }
    // Converting p256dh to a string
    const p256dhArray = Array.from(new Uint8Array(subscription.getKey('p256dh') ?? new ArrayBuffer(0)));
    const p256dhString = btoa(String.fromCharCode.apply(null, p256dhArray));
    // Converting auth to a string
    const authArray = Array.from(new Uint8Array(subscription.getKey('auth') ?? new ArrayBuffer(0)));
    const authString = btoa(String.fromCharCode.apply(null, authArray));
    // Call pushDeviceCreate
    console.log('got subscription', subscription, subscription.getKey('p256dh')?.toString(), subscription.getKey('auth')?.toString());
    documentNodeWrapper<PushDevice, PushDeviceCreateInput>({
        node: pushDeviceCreate,
        input: {
            endpoint: subscription.endpoint,
            expires: subscription.expirationTime ?? undefined,
            keys: {
                auth: authString,
                p256dh: p256dhString,
            }
        },
        successMessage: () => ({ key: 'PushDeviceCreated' }),
        onError: (error) => { PubSub.get().publishSnack({ messageKey: errorToCode(error), severity: 'Error', data: error }); }
    })
};