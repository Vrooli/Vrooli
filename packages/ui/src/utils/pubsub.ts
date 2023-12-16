import { ActiveFocusMode, CommonKey, ErrorKey, Session } from "@local/shared";
import { SnackSeverity } from "components/snacks";

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
    chatMessageEdit: string | false;
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
    theme: "light" | "dark";
    tutorial: void;
}

const defaultPayloads: Partial<{ [K in keyof EventPayloads]: EventPayloads[K] }> = {
    fastUpdate: { on: true, duration: 1000 },
};

export type PubType = keyof EventPayloads;

export class PubSub {
    private static instance: PubSub;
    private subscribers = new Map<PubType, Array<{ token: symbol, subscriber: (data: any) => void }>>();

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }
    static get(): PubSub {
        if (!PubSub.instance) {
            PubSub.instance = new PubSub();
        }
        return PubSub.instance;
    }

    publish<T extends PubType>(type: T, data: EventPayloads[T] = defaultPayloads[type] as EventPayloads[T]) {
        this.subscribers.get(type)?.forEach(({ subscriber }) => subscriber(data));
    }

    subscribe<T extends PubType>(type: T, subscriber: (data: EventPayloads[T]) => void): symbol {
        const token = Symbol(type);

        let subscribers = this.subscribers.get(type);
        if (!subscribers) {
            subscribers = [];
            this.subscribers.set(type, subscribers);
        }
        subscribers.push({ token, subscriber });

        return token;
    }

    unsubscribe(token: symbol) {
        this.subscribers.forEach((subscribers) => {
            const index = subscribers.findIndex(entry => entry.token === token);
            if (index > -1) {
                subscribers.splice(index, 1);
            }
        });
    }
}
