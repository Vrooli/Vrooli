import { AITaskInfo, ActiveFocusMode, ChatPageTabOption, LlmTask, Session, TaskContextInfo, TranslationKeyCommon, TranslationKeyError } from "@local/shared";
import { AlertDialogSeverity } from "components/dialogs/AlertDialog/AlertDialog";
import { SnackSeverity } from "components/snacks/BasicSnack/BasicSnack";
import { ThemeType } from "./localStorage";

export type TranslatedSnackMessage<KeyList = TranslationKeyCommon | TranslationKeyError> = {
    messageKey: KeyList;
    messageVariables?: { [key: string]: string | number };
}
export type UntranslatedSnackMessage = {
    message: string;
}
export type SnackMessage<KeyList = TranslationKeyCommon | TranslationKeyError> = TranslatedSnackMessage<KeyList> | UntranslatedSnackMessage;
export type SnackPub<KeyList = TranslationKeyCommon | TranslationKeyError> = SnackMessage<KeyList> & {
    autoHideDuration?: number | "persist";
    buttonClicked?: (event?: any) => any;
    buttonKey?: TranslationKeyCommon;
    buttonVariables?: { [key: string]: string | number };
    data?: any;
    /**
     * If ID is set, a snack with the same ID will be replaced
     */
    id?: string;
    severity: `${SnackSeverity}`;
};

export type TaskConnect = {
    __type: "taskType";
    /**
     * The task type to link the context to. Will link to the first task of this type.
     */
    data: string;
} | {
    __type: "taskId";
    /**
     * The ID of the task to link the context to
     */
    data: string;
};

export type ContextConnect = TaskConnect | {
    __type: "contextId";
    /**
     * The ID of the context to link the context to
     */
    data: string;
};

/**
 * Provides data to the chat to configure its response behavior and context.
 */
export type ChatTaskPub = {
    /**
     * The ID of the chat containing the tasks and contexts
     */
    chatId: string;
    /**
     * Manages contexts, which provide data to tasks. When a task isn't explicitly set, 
     * we are secretly applying it to the LlmTask.Start task. This is the general context for the chat 
     * without running any specific task/routine.
     */
    contexts?: {
        /**
         * NOTE: Must be serializable to JSON.
         */
        add?: {
            /**
             * Use "replace" to replace all context data for the chat, or "add" to add to the existing
             * context data.
             * 
             * NOTE: This is ignored if connecting using contextId, as it's a one-to-one connection.
             */
            behavior: "replace" | "add";
            /**
             * Used to link the context to a specific task
             */
            connect: ContextConnect;
            value: TaskContextInfo[];
        };
        update?: (Partial<TaskContextInfo> & { id: string })[];
        remove?: ContextConnect[];
    };
    /**
     * Manages tasks in the chat
     */
    tasks?: {
        add?: {
            active?: {
                /**
                * - "force" to set the task regardless of whether there is already an active task. 
                * - "onlyReplaceStart" to set the task only if the current task is not the start task.
                * - "onlyReplaceDifferentTaskType" to set the task only if the task type is different from the current task.
                * 
                * NOTE: Removes the task from the inactive list if it was there.
                */
                behavior: "force" | "onlyReplaceStart" | "onlyReplaceDifferentTaskType";
                value: AITaskInfo | null;
            };
            inactive?: {
                /**
                 * - "onlyIfNoTaskType" to add if the task type is not already suggested or active
                 * - "onlyIfNoTaskId" to add if the task Id is not already suggested or active
                 * - "replaceAll" to replace all suggested tasks
                 * - "replaceTaskType" to replace the first task of the task type. If task was active, it will become inactive.
                 * - "replaceTaskId" to replace the task with the given ID. If task was active, it will become inactive.
                 */
                behavior: "onlyIfNoTaskType" | "onlyIfNoTaskId" | "replaceAll" | "replaceTaskType" | "replaceTaskId";
                value: AITaskInfo[];
            };
        };
        update?: Partial<AITaskInfo>[];
        remove?: TaskConnect[];
    };
}

export type AlertDialogPub = {
    titleKey?: TranslationKeyCommon;
    titleVariables?: { [key: string]: string | number };
    messageKey?: TranslationKeyError | TranslationKeyCommon;
    messageVariables?: { [key: string]: string | number };
    buttons: {
        labelKey: TranslationKeyCommon;
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

export type RequestTaskContextPub = {
    __type: "request";
    chatId: string;
    task: LlmTask | null;
} | {
    __type: "response";
    chatId: string;
    task: LlmTask | null;
    context: TaskContextInfo;
}

export type PopupImagePub = {
    alt: string;
    src: string;
}

export type PopupVideoPub = {
    src: string;
}

/** Determines how many options should be displayed directly in the rich input toolbar */
export type RichInputToolbarViewSize = "minimal" | "partial" | "full";

export const SIDE_MENU_ID = "side-menu" as const;
export const CHAT_SIDE_MENU_ID = "chat-side-menu" as const;
type SideMenuBase = {
    isOpen: boolean;
}
export type SideMenuPayloads = {
    [SIDE_MENU_ID]: SideMenuBase & {
        id: typeof SIDE_MENU_ID;
        /**
         * Optional data to provide to the menu
         */
        data?: {
            isDisplaySettingsCollapsed?: boolean;
            isAdditionalResourcesCollapsed?: boolean;
        };
    };
    [CHAT_SIDE_MENU_ID]: SideMenuBase & {
        id: typeof CHAT_SIDE_MENU_ID;
        /**
         * Optional data to provide to the menu
         */
        data?: {
            /**
             * Tab to switch to
             */
            tab?: ChatPageTabOption | `${ChatPageTabOption}`;
        }
    }
}
export type SideMenuPub = SideMenuPayloads[keyof SideMenuPayloads];

export interface EventPayloads {
    alertDialog: AlertDialogPub;
    /** Can be used to move content based on the appearance of a banner ad */
    banner: { isDisplayed: boolean };
    celebration: CelebrationPub;
    chatTask: ChatTaskPub;
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
    popupImage: PopupImagePub;
    popupVideo: PopupVideoPub;
    requestTaskContext: RequestTaskContextPub;
    richInputToolbarViewSize: RichInputToolbarViewSize;
    session: Session | undefined;
    showBotWarning: boolean;
    sideMenu: SideMenuPub;
    snack: SnackPub;
    theme: ThemeType;
    tutorial: void;
}

const defaultPayloads: Partial<{ [K in keyof EventPayloads]: EventPayloads[K] }> = {
    fastUpdate: { on: true, duration: 1000 },
};

export type PubType = keyof EventPayloads;

export type SubscriberInfo<T> = {
    /** The event handler */
    callback: (data: T) => unknown;
    /** 
     * Any metadata associated with the subscription, 
     * which can be used to identify or filter subscriptions 
     * for advanced use cases.
     */
    metadata?: any;
}


export class PubSub {
    private static instance: PubSub;
    private subscribers = new Map<PubType, Map<symbol, SubscriberInfo<unknown>>>();

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }
    static get(): PubSub {
        if (!PubSub.instance) {
            PubSub.instance = new PubSub();
        }
        return PubSub.instance;
    }

    /**
     * Publishes data to subscribers of a given event type, optionally filtered by a predicate function.
     *
     * @param type - The event type to publish.
     * @param data - The data to send to subscribers. Defaults to an empty payload if not provided.
     * @param filterFn - Optional. A predicate function that takes a subscriber's metadata as input and returns a boolean.
     *                   If provided, only subscribers whose metadata satisfy this function will receive the event.
     */
    publish<T extends PubType>(
        type: T,
        data: EventPayloads[T] = defaultPayloads[type] as EventPayloads[T],
        filterFn?: (metadata: any) => boolean,
    ): void {
        const subscribersOfType = this.subscribers.get(type);
        if (!subscribersOfType) return;

        subscribersOfType.forEach(subscriberInfo => {
            if (!filterFn || filterFn(subscriberInfo.metadata)) {
                subscriberInfo.callback(data);
            }
        });
    }

    /**
     * Subscribe to a given event type
     * @param type The type of event to subscribe to
     * @param subscriber The callback to call when the event is published
     * @param metadata Any metadata to associate with the subscription
     * @returns A function to unsubscribe from the event on hook cleanup
     */
    subscribe<T extends PubType>(
        type: T,
        subscriber: (data: EventPayloads[T]) => void,
        metadata?: any,
    ): () => void {
        const token = Symbol(type);

        if (!this.subscribers.has(type)) {
            this.subscribers.set(type, new Map());
        }
        let subs = this.subscribers.get(type);
        const subscriberInfo: SubscriberInfo<unknown> = {
            callback: subscriber as SubscriberInfo<unknown>["callback"],
            metadata,
        };
        if (subs) {
            subs.set(token, subscriberInfo);
        } else {
            subs = new Map();
            subs.set(token, subscriberInfo);
            this.subscribers.set(type, subs);
        }

        // Return an unsubscribe function
        return () => {
            const subscribersOfType = this.subscribers.get(type);
            if (subscribersOfType) {
                subscribersOfType.delete(token);
                // Clean up the type map if it's empty to prevent memory leaks
                if (subscribersOfType.size === 0) {
                    this.subscribers.delete(type);
                }
            }
        };
    }

    /**
     * Checks whether there are any subscribers for a given event type, optionally filtered by a predicate function.
     *
     * @param type - The event type to check for subscribers.
     * @param filterFn - Optional. A predicate function that takes a subscriber's metadata as input and returns a boolean.
     *                   If provided, only subscribers whose metadata satisfy this function are considered.
     * @returns True if there is at least one subscriber for the event type that matches the filter (if provided); otherwise, false.
     */
    hasSubscribers<T extends PubType>(type: T, filterFn?: (metadata: any) => boolean): boolean {
        const subs = this.subscribers.get(type);
        if (!subs || subs.size === 0) return false;

        if (!filterFn) {
            return subs.size > 0;
        }

        for (const subscriberInfo of subs.values()) {
            if (filterFn(subscriberInfo.metadata)) {
                return true;
            }
        }
        return false;
    }
}
