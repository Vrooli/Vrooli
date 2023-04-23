export class PubSub {
    static instance;
    subscribers = {};
    constructor() { }
    static get() {
        if (!PubSub.instance) {
            PubSub.instance = new PubSub();
        }
        return PubSub.instance;
    }
    publish(key, data) {
        if (this.subscribers[key]) {
            this.subscribers[key].forEach(subscriber => subscriber[1](data));
        }
    }
    publishAlertDialog(data) {
        this.publish("AlertDialog", data);
    }
    publishCelebration(duration) {
        this.publish("Celebration", duration);
    }
    publishCommandPalette() {
        this.publish("CommandPalette");
    }
    publishCookies() {
        this.publish("Cookies");
    }
    publishFastUpdate({ on = true, duration = 1000 }) {
        this.publish("FastUpdate", { on, duration });
    }
    publishFindInPage() {
        this.publish("FindInPage");
    }
    publishFocusMode(mode) {
        this.publish("FocusMode", mode);
    }
    publishFontSize(fontSize) {
        this.publish("FontSize", fontSize);
    }
    publishIsLeftHanded(isLeftHanded) {
        this.publish("IsLeftHanded", isLeftHanded);
    }
    publishLanguage(language) {
        this.publish("Language", language);
    }
    publishLoading(spinnerDelay) {
        this.publish("Loading", spinnerDelay);
    }
    publishLogOut() {
        this.publish("LogOut");
    }
    publishNodeDrag(data) {
        this.publish("NodeDrag", data);
    }
    publishNodeDrop(data) {
        this.publish("NodeDrop", data);
    }
    publishSession(session) {
        localStorage.setItem("isLoggedIn", session?.isLoggedIn === true ? "true" : "false");
        this.publish("Session", session);
    }
    publishSnack(data) {
        this.publish("Snack", data);
    }
    publishTheme(theme) {
        this.publish("Theme", theme);
    }
    publishWelcome() {
        this.publish("Welcome");
    }
    subscribe(key, subscriber) {
        const token = Symbol(key);
        if (!this.subscribers[key]) {
            this.subscribers[key] = [];
        }
        this.subscribers[key].push([token, subscriber]);
        return token;
    }
    subscribeAlertDialog(subscriber) {
        return this.subscribe("AlertDialog", subscriber);
    }
    subscribeCelebration(subscriber) {
        return this.subscribe("Celebration", subscriber);
    }
    subscribeCommandPalette(subscriber) {
        return this.subscribe("CommandPalette", subscriber);
    }
    subscribeCookies(subscriber) {
        return this.subscribe("Cookies", subscriber);
    }
    subscribeFastUpdate(subscriber) {
        return this.subscribe("FastUpdate", subscriber);
    }
    subscribeFindInPage(subscriber) {
        return this.subscribe("FindInPage", subscriber);
    }
    subscribeFocusMode(subscriber) {
        return this.subscribe("FocusMode", subscriber);
    }
    subscribeFontSize(subscriber) {
        return this.subscribe("FontSize", subscriber);
    }
    subscribeIsLeftHanded(subscriber) {
        return this.subscribe("IsLeftHanded", subscriber);
    }
    subscribeLanguage(subscriber) {
        return this.subscribe("Language", subscriber);
    }
    subscribeLoading(subscriber) {
        return this.subscribe("Loading", subscriber);
    }
    subscribeLogOut(subscriber) {
        return this.subscribe("LogOut", subscriber);
    }
    subscribeNodeDrag(subscriber) {
        return this.subscribe("NodeDrag", subscriber);
    }
    subscribeNodeDrop(subscriber) {
        return this.subscribe("NodeDrop", subscriber);
    }
    subscribeSession(subscriber) {
        return this.subscribe("Session", subscriber);
    }
    subscribeSnack(subscriber) {
        return this.subscribe("Snack", subscriber);
    }
    subscribeTheme(subscriber) {
        return this.subscribe("Theme", subscriber);
    }
    subscribeWelcome(subscriber) {
        return this.subscribe("Welcome", subscriber);
    }
    unsubscribe(token) {
        for (const key in this.subscribers) {
            const subscribers = this.subscribers[key];
            const index = subscribers.findIndex(subscriber => subscriber[0] === token);
            if (index > -1) {
                subscribers.splice(index, 1);
            }
        }
    }
}
//# sourceMappingURL=pubsub.js.map