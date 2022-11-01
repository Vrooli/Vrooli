/**
 * Simple pub/sub implementation, with typescript support. 
 * Uses a singleton instance to publish and subscribe to events.
 * example:
 *      import { PubSub } from 'utils';
 *      PubSub.get().publishSnack({ message: 'Hello World' });
 */
import { COOKIE, ValueOf } from '@shared/consts';
import { AlertDialogState, SnackSeverity } from 'components';
import { Session } from 'types';

export const Pubs = {
    ...COOKIE,
    Celebration: "celebration",
    CommandPalette: "commandPalette",
    Cookies: "cookies", // For cookie consent dialog
    FastUpdate: "fastUpdate",
    FindInPage: "findInPage",
    Loading: "loading",
    LogOut: "logout",
    AlertDialog: "alertDialog",
    Session: "session",
    Snack: "snack",
    Theme: "theme",
    NodeDrag: "NodeDrag",
    NodeDrop: "NodeDrop",
}
export type Pubs = ValueOf<typeof Pubs>;

export type SnackPub = {
    autoHideDuration?: number | 'persist';
    buttonClicked?: (event?: any) => any;
    buttonText?: string;
    data?: any;
    /**
     * If ID is set, a snack with the same ID will be replaced
     */
    id?: string;
    message?: string;
    severity: SnackSeverity;
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
    publishCelebration(duration?: number) {
        this.publish(Pubs.Celebration, duration);
    }
    publishCommandPalette() {
        this.publish(Pubs.CommandPalette);
    }
    publishCookies() {
        this.publish(Pubs.Cookies);
    }
    publishFindInPage() {
        this.publish(Pubs.FindInPage);
    }
    /**
     * Notifies graph links to re-render quickly for a period of time
     */
    publishFastUpdate({ on = true, duration = 1000 }: { on?: boolean, duration?: number }) {
        this.publish(Pubs.FastUpdate, { on, duration });
    }
    /**
     * Pass delay to show spinner if turning on, or false to turn off.
     */
    publishLoading(spinnerDelay: number | false) {
        this.publish(Pubs.Loading, spinnerDelay);
    }
    publishLogOut() {
        this.publish(Pubs.LogOut);
    }
    publishAlertDialog(data: AlertDialogState) {
        this.publish(Pubs.AlertDialog, data);
    }
    publishSession(session: Session | undefined) {
        this.publish(Pubs.Session, session);
    }
    publishSnack(data: SnackPub) {
        this.publish(Pubs.Snack, data);
    }
    publishTheme(theme: 'light' | 'dark') {
        this.publish(Pubs.Theme, theme);
    }
    publishNodeDrag(data: { nodeId: string }) {
        this.publish(Pubs.NodeDrag, data);
    }
    publishNodeDrop(data: { nodeId: string, position: { x: number, y: number } }) {
        this.publish(Pubs.NodeDrop, data);
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
    subscribeCelebration(subscriber: (duration?: number) => void) {
        return this.subscribe(Pubs.Celebration, subscriber);
    }
    subscribeCommandPalette(subscriber: () => void) {
        return this.subscribe(Pubs.CommandPalette, subscriber);
    }
    subscribeCookies(subscriber: () => void) {
        return this.subscribe(Pubs.Cookies, subscriber);
    }
    subscribeFindInPage(subscriber: () => void) {
        return this.subscribe(Pubs.FindInPage, subscriber);
    }
    subscribeFastUpdate(subscriber: ({ on, duration }: { on: boolean, duration: number }) => void) {
        return this.subscribe(Pubs.FastUpdate, subscriber);
    }
    subscribeLoading(subscriber: (spinnerDelay: number | false) => void) {
        return this.subscribe(Pubs.Loading, subscriber);
    }
    subscribeLogOut(subscriber: () => void) {
        return this.subscribe(Pubs.LogOut, subscriber);
    }
    subscribeAlertDialog(subscriber: (data: AlertDialogState) => void) {
        return this.subscribe(Pubs.AlertDialog, subscriber);
    }
    subscribeSession(subscriber: (session: Session | undefined) => void) {
        return this.subscribe(Pubs.Session, subscriber);
    }
    subscribeSnack(subscriber: (data: SnackPub) => void) {
        return this.subscribe(Pubs.Snack, subscriber);
    }
    subscribeTheme(subscriber: (theme: 'light' | 'dark') => void) {
        return this.subscribe(Pubs.Theme, subscriber);
    }
    subscribeNodeDrag(subscriber: (data: { nodeId: string }) => void) {
        return this.subscribe(Pubs.NodeDrag, subscriber);
    }
    subscribeNodeDrop(subscriber: (data: { nodeId: string, position: { x: number, y: number } }) => void) {
        return this.subscribe(Pubs.NodeDrop, subscriber);
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
