export const i18nextTMock = (key, _options) => {
    // Error key with details
    if (key === "CannotConnectToServer")
        return "Cannot connect to server";
    if (key === "CannotConnectToServerDetails")
        return "The details of cannot connect to server";
    // Common key without details
    if (key === "ChangePassword")
        return "Change password";
    return key;
};

const i18nextMock = {
    // Includes some basic keys to test with
    t: jest.fn(i18nextTMock),
};

export default i18nextMock;
