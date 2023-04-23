import { LINKS } from "@local/consts";
import { isOfType } from "@local/utils";
import { adaHandleRegex, urlRegex, walletAddressRegex } from "@local/validation";
import { ResourceType } from "../consts";
import { stringifySearchParams } from "../route";
import { uuidToBase36 } from "./urlTools";
export const getObjectUrlBase = (object) => {
    if (isOfType(object, "User"))
        return LINKS.Profile;
    if (isOfType(object, "Bookmark", "Reaction", "View"))
        return getObjectUrlBase(object.to);
    if (isOfType(object, "RunRoutine"))
        return getObjectUrlBase(object.routineVersion);
    if (isOfType(object, "RunProject"))
        return getObjectUrlBase(object.projectVersion);
    return LINKS[object.__typename.replace("Version", "")];
};
export const getObjectSlug = (object) => {
    if (isOfType(object, "Action", "Shortcut", "CalendarEvent"))
        return "";
    if (isOfType(object, "Bookmark", "Reaction", "View"))
        return getObjectSlug(object.to);
    if (isOfType(object, "RunRoutine"))
        return getObjectSlug(object.routineVersion);
    if (isOfType(object, "RunProject"))
        return getObjectSlug(object.projectVersion);
    if (object.root)
        return object.root.handle ?? uuidToBase36(object.root.id);
    return object.handle ?? uuidToBase36(object.id);
};
export const getObjectSearchParams = (object) => {
    if (isOfType(object, "Action", "Shortcut"))
        return "";
    if (isOfType(object, "CalendarEvent"))
        return stringifySearchParams({ start: object.start });
    if (object.__typename === "RunRoutine")
        return stringifySearchParams({ run: uuidToBase36(object.id) });
    return "";
};
export const getObjectUrl = (object) => isOfType(object, "Action") ? "" :
    isOfType(object, "Shortcut", "CalendarEvent") ? object.id :
        `${getObjectUrlBase(object)}/${getObjectSlug(object)}${getObjectSearchParams(object)}`;
export const openObject = (object, setLocation) => !isOfType(object, "Action") && setLocation(getObjectUrl(object));
export const getObjectEditUrl = (object) => `${getObjectUrlBase(object)}/edit/${getObjectSlug(object)}${getObjectSearchParams(object)}`;
export const openObjectEdit = (object, setLocation) => setLocation(getObjectEditUrl(object));
export const getObjectReportUrl = (object) => `${getObjectUrlBase(object)}/reports/${getObjectSlug(object)}`;
export const openObjectReport = (object, setLocation) => setLocation(getObjectReportUrl(object));
export const getResourceType = (link) => {
    if (urlRegex.test(link))
        return ResourceType.Url;
    if (walletAddressRegex.test(link))
        return ResourceType.Wallet;
    if (adaHandleRegex.test(link))
        return ResourceType.Handle;
    return null;
};
export const getResourceUrl = (link) => {
    const resourceType = getResourceType(link);
    if (resourceType === ResourceType.Url)
        return link;
    if (resourceType === ResourceType.Handle)
        return `https://handle.me/${link}`;
    if (resourceType === ResourceType.Wallet)
        return `https://cardanoscan.io/address/${link}`;
    return undefined;
};
//# sourceMappingURL=openObject.js.map