/**
 * Simple pub/sub implementation, with typescript support. 
 * Uses a singleton instance to publish and subscribe to events.
 * example:
 *      import { PubSub } from "../utils";
 *      PubSub.get().publishSnack({ messageKey: 'HelloWorld' });
 */
import { ActiveFocusMode, Session } from "@local/consts";
import { CommonKey, ErrorKey } from "@local/translations";
import { SnackSeverity } from "../components/dialogs/snacks";

export type Pubs = "Celebration" |
    "CommandPalette" |
    "Cookies" | // For cookie consent dialog
    "FastUpdate" |
    "FindInPage" |
    "FocusMode" |
    "FontSize" |
    "IsLeftHanded" |
    "Language" |
    "Loading" |
    "LogOut" |
    "AlertDialog" |
    "Session" |
    "Snack" |
    "Theme" |
    "NodeDrag" |
    "NodeDrop" |
    "Welcome";


export type TranslatedSnackMessage = {
    messageKey: ErrorKey | CommonKey;
    messageVariables?: { [key: string]: string | number };
}
export type UntranslatedSnackMessage = {
    message: string;
}
export type SnackMessage = TranslatedSnackMessage | UntranslatedSnackMessage;
export type SnackPub = SnackMessage & {
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
        onClick?: (() => void);
    }[];
}

export class PubSub {
    private static instance: PubSub;
    private subscribers: { [key: string]: [symbol, Function][] } = {};
    private constructor() { }
    static get(): PubSub {
        if (!PubSub.instance) {
            PubSub.instance = new PubSub();
        }
        return PubSub.instance;
    }

    publish(key: Pubs, data?: any) {
        if (this.subscribers[key]) {
            this.subscribers[key].forEach(subscriber => subscriber[1](data));
        }
    }
    publishAlertDialog(data: AlertDialogPub) {
        this.publish("AlertDialog", data);
    }
    publishCelebration(duration?: number) {
        this.publish("Celebration", duration);
    }
    publishCommandPalette() {
        this.publish("CommandPalette");
    }
    publishCookies() {
        this.publish("Cookies");
    }
    /**
     * Notifies graph links to re-render quickly for a period of time
     */
    publishFastUpdate({ on = true, duration = 1000 }: { on?: boolean, duration?: number }) {
        this.publish("FastUpdate", { on, duration });
    }
    publishFindInPage() {
        this.publish("FindInPage");
    }
    publishFocusMode(mode: ActiveFocusMode) {
        this.publish("FocusMode", mode);
    }
    publishFontSize(fontSize: number) {
        this.publish("FontSize", fontSize);
    }
    publishIsLeftHanded(isLeftHanded: boolean) {
        this.publish("IsLeftHanded", isLeftHanded);
    }
    publishLanguage(language: string) {
        this.publish("Language", language);
    }
    /**
     * Pass delay to show spinner if turning on, or false to turn off.
     */
    publishLoading(spinnerDelay: number | false) {
        this.publish("Loading", spinnerDelay);
    }
    publishLogOut() {
        this.publish("LogOut");
    }
    publishNodeDrag(data: { nodeId: string }) {
        this.publish("NodeDrag", data);
    }
    publishNodeDrop(data: { nodeId: string, position: { x: number, y: number } }) {
        this.publish("NodeDrop", data);
    }
    publishSession(session: Session | undefined) {
        // When session is published, also set "isLoggedIn" flag in localStorage
        localStorage.setItem("isLoggedIn", session?.isLoggedIn === true ? "true" : "false");
        this.publish("Session", session);
    }
    publishSnack(data: SnackPub) {
        this.publish("Snack", data);
    }
    publishTheme(theme: "light" | "dark") {
        this.publish("Theme", theme);
    }
    publishWelcome() {
        this.publish("Welcome");
    }

    subscribe(key: Pubs, subscriber: Function): symbol {
        // Create unique token, so we can unsubscribe later
        const token = Symbol(key);
        if (!this.subscribers[key]) {
            this.subscribers[key] = [];
        }
        this.subscribers[key].push([token, subscriber]);
        return token;
    }
    subscribeAlertDialog(subscriber: (data: AlertDialogPub) => void) {
        return this.subscribe("AlertDialog", subscriber);
    }
    subscribeCelebration(subscriber: (duration?: number) => void) {
        return this.subscribe("Celebration", subscriber);
    }
    subscribeCommandPalette(subscriber: () => void) {
        return this.subscribe("CommandPalette", subscriber);
    }
    subscribeCookies(subscriber: () => void) {
        return this.subscribe("Cookies", subscriber);
    }
    subscribeFastUpdate(subscriber: ({ on, duration }: { on: boolean, duration: number }) => void) {
        return this.subscribe("FastUpdate", subscriber);
    }
    subscribeFindInPage(subscriber: () => void) {
        return this.subscribe("FindInPage", subscriber);
    }
    subscribeFocusMode(subscriber: (mode: ActiveFocusMode) => void) {
        return this.subscribe("FocusMode", subscriber);
    }
    subscribeFontSize(subscriber: (fontSize: number) => void) {
        return this.subscribe("FontSize", subscriber);
    }
    subscribeIsLeftHanded(subscriber: (isLeftHanded: boolean) => void) {
        return this.subscribe("IsLeftHanded", subscriber);
    }
    subscribeLanguage(subscriber: (language: string) => void) {
        return this.subscribe("Language", subscriber);
    }
    subscribeLoading(subscriber: (spinnerDelay: number | false) => void) {
        return this.subscribe("Loading", subscriber);
    }
    subscribeLogOut(subscriber: () => void) {
        return this.subscribe("LogOut", subscriber);
    }
    subscribeNodeDrag(subscriber: (data: { nodeId: string }) => void) {
        return this.subscribe("NodeDrag", subscriber);
    }
    subscribeNodeDrop(subscriber: (data: { nodeId: string, position: { x: number, y: number } }) => void) {
        return this.subscribe("NodeDrop", subscriber);
    }
    subscribeSession(subscriber: (session: Session | undefined) => void) {
        return this.subscribe("Session", subscriber);
    }
    subscribeSnack(subscriber: (data: SnackPub) => void) {
        return this.subscribe("Snack", subscriber);
    }
    subscribeTheme(subscriber: (theme: "light" | "dark") => void) {
        return this.subscribe("Theme", subscriber);
    }
    subscribeWelcome(subscriber: () => void) {
        return this.subscribe("Welcome", subscriber);
    }

    unsubscribe(token: symbol) {
        for (const key in this.subscribers) {
            const subscribers = this.subscribers[key];
            const index = subscribers.findIndex(subscriber => subscriber[0] === token);
            if (index > -1) {
                subscribers.splice(index, 1);
            }
        }
    }
}
