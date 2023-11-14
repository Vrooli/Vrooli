const mockInstance = {
    publish: jest.fn(),
    publishAlertDialog: jest.fn(),
    publishCelebration: jest.fn(),
    publishCommandPalette: jest.fn(),
    publishCookies: jest.fn(),
    publishFastUpdate: jest.fn(),
    publishFindInPage: jest.fn(),
    publishFocusMode: jest.fn(),
    publishFontSize: jest.fn(),
    publishIsLeftHanded: jest.fn(),
    publishLanguage: jest.fn(),
    publishLoading: jest.fn(),
    publishLogOut: jest.fn(),
    publishNodeDrag: jest.fn(),
    publishNodeDrop: jest.fn(),
    publishSession: jest.fn(),
    publishSideMenu: jest.fn(),
    publishSnack: jest.fn(),
    publishTheme: jest.fn(),
    publishTutorial: jest.fn(),

    subscribe: jest.fn(),
    subscribeAlertDialog: jest.fn(),
    subscribeCelebration: jest.fn(),
    subscribeCommandPalette: jest.fn(),
    subscribeCookies: jest.fn(),
    subscribeFastUpdate: jest.fn(),
    subscribeFindInPage: jest.fn(),
    subscribeFocusMode: jest.fn(),
    subscribeFontSize: jest.fn(),
    subscribeIsLeftHanded: jest.fn(),
    subscribeLanguage: jest.fn(),
    subscribeLoading: jest.fn(),
    subscribeLogOut: jest.fn(),
    subscribeNodeDrag: jest.fn(),
    subscribeNodeDrop: jest.fn(),
    subscribeSession: jest.fn(),
    subscribeSideMenu: jest.fn(),
    subscribeSnack: jest.fn(),
    subscribeTheme: jest.fn(),
    subscribeTutorial: jest.fn(),

    unsubscribe: jest.fn(),
};

export class PubSub {
    // Override the get method to return the mock instance
    static get() {
        return mockInstance;
    }
}
