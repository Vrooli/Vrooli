import { ActiveFocusMode, CommonKey, ErrorKey, Session } from "@local/shared";
import { AlertDialogSeverity } from "components/dialogs/AlertDialog/AlertDialog";
import { SnackSeverity } from "components/snacks";
import { ThemeType } from "./cookies";

export type TranslatedSnackMessage<KeyList = CommonKey | ErrorKey> = {
    messageKey: KeyList;
    messageVariables?: { [key: string]: string | number };
}
export type UntranslatedSnackMessage = {
    message: string;
}
export type SnackMessage<KeyList = CommonKey | ErrorKey> = TranslatedSnackMessage<KeyList> | UntranslatedSnackMessage;
export type SnackPub<KeyList = CommonKey | ErrorKey> = SnackMessage<KeyList> & {
    autoHideDuration?: number | "persist";
    buttonClicked?: (event?: any) => any;
    buttonKey?: CommonKey;
    buttonVariables?: { [key: string]: string | number };
    data?: any;
    /**
     * If ID is set, a snack with the same ID will be replaced
     */
    id?: string;
    severity: `${SnackSeverity}`;
};

export type AlertDialogPub = {
    titleKey?: CommonKey;
    titleVariables?: { [key: string]: string | number };
    messageKey?: ErrorKey | CommonKey;
    messageVariables?: { [key: string]: string | number };
    buttons: {
        labelKey: CommonKey;
        labelVariables?: { [key: string]: string | number };
        onClick?: (() => unknown);
    }[];
    severity?: `${AlertDialogSeverity}`;
}

export type CelebrationType = "balloons" | "confetti" | "emoji";
export type CelebrationPub = {
    celebrationType?: CelebrationType;
    duration?: number;
    emojis?: string[];
    targetId?: string;
}

export type SideMenuPub = {
    id: "side-menu" | "chat-side-menu";
    idPrefix?: string;
    isOpen: boolean;
}

export interface EventPayloads {
    alertDialog: AlertDialogPub;
    celebration: CelebrationPub;
    commandPalette: void;
    cookies: void;
    fastUpdate: { on?: boolean, duration?: number };
    findInPage: void;
    focusMode: ActiveFocusMode;
    fontSize: number;
    isLeftHanded: boolean;
    language: string;
    /** Pass delay to show spinner if turning on, or false to turn off. */
    loading: number | boolean;
    logOut: void;
    nodeDrag: { nodeId: string };
    nodeDrop: { nodeId: string, position: { x: number, y: number } };
    session: Session | undefined;
    sideMenu: SideMenuPub;
    snack: SnackPub;
    theme: ThemeType;
    tutorial: void;
}

const defaultPayloads: Partial<{ [K in keyof EventPayloads]: EventPayloads[K] }> = {
    fastUpdate: { on: true, duration: 1000 },
};

export type PubType = keyof EventPayloads;

export class PubSub {
    private static instance: PubSub;
    private subscribers = new Map<PubType, Map<symbol, (data: unknown) => void>>();

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }
    static get(): PubSub {
        if (!PubSub.instance) {
            PubSub.instance = new PubSub();
        }
        return PubSub.instance;
    }

    /**
     * Publish data to all subscribers of a given type
     * @param type The type of event to publish
     * @param data The data to publish to subscribers
     */
    publish<T extends PubType>(type: T, data: EventPayloads[T] = defaultPayloads[type] as EventPayloads[T]) {
        this.subscribers.get(type)?.forEach(subscriber => subscriber(data));
    }

    /**
     * Subscribe to a given event type
     * @param type The type of event to subscribe to
     * @param subscriber The callback to call when the event is published
     * @returns A function to unsubscribe from the event on hook cleanup
     */
    subscribe<T extends PubType>(type: T, subscriber: (data: EventPayloads[T]) => void): () => void {
        const token = Symbol(type);

        if (!this.subscribers.has(type)) {
            this.subscribers.set(type, new Map());
        }
        let subs = this.subscribers.get(type);
        if (subs) {
            subs.set(token, subscriber as (data: unknown) => void);
        } else {
            subs = new Map();
            subs.set(token, subscriber as (data: unknown) => void);
            this.subscribers.set(type, subs);
        }

        // Return an unsubscribe function
        return () => {
            const subscribersOfType = this.subscribers.get(type);
            if (subscribersOfType) {
                subscribersOfType.delete(token);
                // // Optionally, clean up the type map if it's empty to prevent memory leaks
                // if (subscribersOfType.size === 0) {
                //     this.subscribers.delete(type);
                // }
            }
        };
    }
}
