import { LINKS } from "@local/consts";
import { uuidValidate } from "@local/uuid";
import { adaHandleRegex } from "@local/validation";
import { PubSub } from "../pubsub";
import { getLastUrlPart } from "../route";
function toBigInt(value, radix) {
    return [...value.toString()]
        .reduce((r, v) => r * BigInt(radix) + BigInt(parseInt(v, radix)), 0n);
}
export const uuidToBase36 = (uuid) => {
    try {
        const base36 = toBigInt(uuid.replace(/-/g, ""), 16).toString(36);
        return base36 === "0" ? "" : base36;
    }
    catch (error) {
        PubSub.get().publishSnack({ messageKey: "CouldNotConvertId", severity: "Error", data: { uuid } });
        return "";
    }
};
export const base36ToUuid = (base36, showError = true) => {
    try {
        const uuid = toBigInt(base36, 36).toString(16).padStart(32, "0").replace(/(.{8})(.{4})(.{4})(.{4})(.{12})/, "$1-$2-$3-$4-$5");
        return uuid === "0" ? "" : uuid;
    }
    catch (error) {
        if (showError)
            PubSub.get().publishSnack({ messageKey: "InvalidUrlId", severity: "Error", data: { base36 } });
        return "";
    }
};
export const parseSingleItemUrl = () => {
    const returnObject = {};
    const lastPart = getLastUrlPart();
    const secondLastPart = getLastUrlPart(1);
    if (adaHandleRegex.test(secondLastPart)) {
        returnObject.handleRoot = secondLastPart;
    }
    else if (uuidValidate(base36ToUuid(secondLastPart, false))) {
        returnObject.idRoot = base36ToUuid(secondLastPart);
    }
    const objectsWithVersions = [
        LINKS.Api,
        LINKS.Note,
        LINKS.Project,
        LINKS.Routine,
        LINKS.SmartContract,
        LINKS.Standard,
    ].map(link => link.split("/").pop());
    const allUrlParts = window.location.pathname.split("/");
    const isVersioned = allUrlParts.some(part => objectsWithVersions.includes(part));
    if (adaHandleRegex.test(lastPart)) {
        if (isVersioned)
            returnObject.handleRoot = lastPart;
        else
            returnObject.handle = lastPart;
    }
    else if (uuidValidate(base36ToUuid(lastPart, false))) {
        if (isVersioned)
            returnObject.idRoot = base36ToUuid(lastPart, false);
        else
            returnObject.id = base36ToUuid(lastPart, false);
    }
    console.log("parseSingleItemUrl RESULT", returnObject);
    return returnObject;
};
//# sourceMappingURL=urlTools.js.map