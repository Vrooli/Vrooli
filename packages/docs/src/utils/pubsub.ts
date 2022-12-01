/**
 * Simple pub/sub implementation, with typescript support. 
 * Uses a singleton instance to publish and subscribe to events.
 */
import { AlertDialogState } from 'components';

export type Pubs = 'Celebration' | 
    'Loading' | 
    'AlertDialog' |
    'Theme' | 
    'NodeDrag' | 
    'NodeDrop';

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
        this.publish('Celebration', duration);
    }
    /**
     * Pass delay to show spinner if turning on, or false to turn off.
     */
    publishLoading(spinnerDelay: number | false) {
        this.publish('Loading', spinnerDelay);
    }
    publishAlertDialog(data: AlertDialogState) {
        this.publish('AlertDialog', data);
    }
    publishTheme(theme: 'light' | 'dark') {
        this.publish('Theme', theme);
    }
    publishNodeDrag(data: { nodeId: string }) {
        this.publish('NodeDrag', data);
    }
    publishNodeDrop(data: { nodeId: string, position: { x: number, y: number } }) {
        this.publish('NodeDrop', data);
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
        return this.subscribe('Celebration', subscriber);
    }
    subscribeLoading(subscriber: (spinnerDelay: number | false) => void) {
        return this.subscribe('Loading', subscriber);
    }
    subscribeAlertDialog(subscriber: (data: AlertDialogState) => void) {
        return this.subscribe('AlertDialog', subscriber);
    }
    subscribeNodeDrag(subscriber: (data: { nodeId: string }) => void) {
        return this.subscribe('NodeDrag', subscriber);
    }
    subscribeNodeDrop(subscriber: (data: { nodeId: string, position: { x: number, y: number } }) => void) {
        return this.subscribe('NodeDrop', subscriber);
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
