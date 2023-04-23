import { adaHandleRegex, urlRegex, walletAddressRegex } from "../regex";
export const addHttps = (value) => {
    if (typeof value === "string" &&
        !value.startsWith("http://") &&
        !value.startsWith("https://") &&
        !walletAddressRegex.test(value) &&
        !adaHandleRegex.test(value) &&
        urlRegex.test(`https://${value}`)) {
        return `https://${value}`;
    }
    return value;
};
//# sourceMappingURL=addHttps.js.map