const i18nextMock = {
    // Includes some basic keys to test with
    t: jest.fn((key, _options) => {
        console.log("i18next.t called with key", key);
        // Error key with details
        if (key === "CannotConnectToServer") return "Cannot connect to server";
        if (key === "CannotConnectToServerDetails") return "The details of cannot connect to server";
        // Common key without details
        if (key === "ChangePassword") return "Change password";
        return key;
    }),
};

export default i18nextMock;
