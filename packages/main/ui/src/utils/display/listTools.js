import { jsx as _jsx } from "react/jsx-runtime";
import { exists, isOfType } from "@local/utils";
import { ObjectListItem } from "../../components/lists/ObjectListItem/ObjectListItem";
import { valueFromDot } from "../shape/general";
import { displayDate, firstString } from "./stringTools";
import { getTranslation, getUserLanguages } from "./translationTools";
export const getYouDot = (object, property) => {
    if (!object)
        return null;
    if (isOfType(object, "Bookmark", "View", "Vote"))
        return getYouDot(object.to, property);
    if (isOfType(object, "RunRoutine"))
        return getYouDot(object.routineVersion, property);
    if (isOfType(object, "RunProject"))
        return getYouDot(object.projectVersion, property);
    if (exists(object.you?.[property]))
        return `you.${property}`;
    if (exists(object.root?.you?.[property]))
        return `root.you.${property}`;
    return null;
};
export const defaultYou = {
    canComment: false,
    canCopy: false,
    canDelete: false,
    canRead: false,
    canReport: false,
    canShare: false,
    canBookmark: false,
    canUpdate: false,
    canReact: false,
    isBookmarked: false,
    isViewed: false,
    reaction: null,
};
export const getYou = (object) => {
    const defaultPermissions = { ...defaultYou };
    if (!object)
        return defaultPermissions;
    if (isOfType(object, "Bookmark", "View", "Vote"))
        return getYou(object.to);
    if (isOfType(object, "RunRoutine"))
        return getYou(object.routineVersion);
    if (isOfType(object, "RunProject"))
        return getYou(object.projectVersion);
    for (const key in defaultPermissions) {
        const field = valueFromDot(object, `you.${key}`);
        if (field === true || field === false || typeof field === "string")
            defaultPermissions[key] = field;
        else {
            const field = valueFromDot(object, `root.you.${key}`);
            if (field === true || field === false || typeof field === "string")
                defaultPermissions[key] = field;
        }
    }
    return defaultPermissions;
};
export const getCounts = (object) => {
    const defaultCounts = {
        comments: 0,
        forks: 0,
        issues: 0,
        labels: 0,
        pullRequests: 0,
        questions: 0,
        reports: 0,
        score: 0,
        bookmarks: 0,
        transfers: 0,
        translations: 0,
        versions: 0,
        views: 0,
    };
    if (!object)
        return defaultCounts;
    if (isOfType(object, "Bookmark", "View", "Vote"))
        return getCounts(object.to);
    if (isOfType(object, "RunRoutine"))
        return getCounts(object.routineVersion);
    if (isOfType(object, "RunProject"))
        return getCounts(object.projectVersion);
    if (isOfType(object, "NodeRoutineListItem"))
        return getCounts(object.routineVersion);
    for (const key in defaultCounts) {
        const objectProp = ["score", "views"].includes(key) ? key : `${key}Count`;
        const field = valueFromDot(object, objectProp);
        if (field !== undefined)
            defaultCounts[key] = field;
        else {
            const field = valueFromDot(object, `root.${objectProp}`);
            if (field !== undefined)
                defaultCounts[key] = field;
        }
    }
    return defaultCounts;
};
const tryTitle = (obj, langs) => {
    const translations = getTranslation(obj, langs, true);
    return firstString(obj.title, obj.name, obj.label, translations.title, translations.name, translations.label, obj.handle ? `$${obj.handle}` : null);
};
const trySubtitle = (obj, langs) => {
    const translations = getTranslation(obj, langs, true);
    return firstString(obj.bio, obj.description, obj.summary, obj.details, obj.text, translations.bio, translations.description, translations.summary, translations.details, translations.text);
};
const tryVersioned = (obj, langs) => {
    let title = null;
    let subtitle = null;
    const objectsToCheck = [
        obj,
        obj.root,
        obj.versions?.find(v => v.isLatest),
        ...([...(obj.versions ?? [])].sort((a, b) => b.versionIndex - a.versionIndex)),
    ];
    for (const curr of objectsToCheck) {
        if (!exists(curr))
            continue;
        title = tryTitle(curr, langs);
        subtitle = trySubtitle(curr, langs);
        if (title && subtitle)
            break;
    }
    return { title: title ?? "", subtitle: subtitle ?? "" };
};
export const getDisplay = (object, languages) => {
    if (!object)
        return { title: "", subtitle: "" };
    if (isOfType(object, "Bookmark", "View", "Vote"))
        return getDisplay(object.to);
    const langs = languages ?? getUserLanguages(undefined);
    if (isOfType(object, "RunRoutine")) {
        const { completedAt, name, routineVersion, startedAt } = object;
        const title = firstString(name, getTranslation(routineVersion, langs, true).name);
        const started = startedAt ? displayDate(startedAt) : null;
        const completed = completedAt ? displayDate(completedAt) : null;
        return {
            title: started ? `${title} (started)` : title,
            subtitle: started ? "Started: " + started : completed ? "Completed: " + completed : "",
        };
    }
    if (isOfType(object, "RunProject")) {
        const { completedAt, name, projectVersion, startedAt } = object;
        const title = firstString(name, getTranslation(projectVersion, langs, true).name);
        const started = startedAt ? displayDate(startedAt) : null;
        const completed = completedAt ? displayDate(completedAt) : null;
        return {
            title: started ? `${title} (started)` : title,
            subtitle: started ? "Started: " + started : completed ? "Completed: " + completed : "",
        };
    }
    if (isOfType(object, "Member"))
        return getDisplay(object.user);
    const { title, subtitle } = tryVersioned(object, langs);
    if (isOfType(object, "NodeRoutineListItem") && title.length === 0 && subtitle.length === 0) {
        const routineVersionDisplay = getDisplay(object.routineVersion, languages);
        return {
            title: title.length === 0 ? routineVersionDisplay.title : title,
            subtitle: subtitle.length === 0 ? routineVersionDisplay.subtitle : subtitle,
        };
    }
    return { title, subtitle };
};
export const getBookmarkFor = (object) => {
    if (!object)
        return { bookmarkFor: null, starForId: null };
    if (isOfType(object, "BookmarkList", "Member"))
        return { bookmarkFor: null, starForId: null };
    if (isOfType(object, "Bookmark", "View", "Vote"))
        return getBookmarkFor(object.to);
    if (isOfType(object, "RunRoutine"))
        return getBookmarkFor(object.routineVersion);
    if (isOfType(object, "RunProject"))
        return getBookmarkFor(object.projectVersion);
    if (isOfType(object, "NodeRoutineListItem"))
        return getBookmarkFor(object.routineVersion);
    if (object.root)
        return getBookmarkFor(object.root);
    return { bookmarkFor: object.__typename, starForId: object.id };
};
export function listToAutocomplete(objects, languages) {
    return objects.map(o => ({
        __typename: o.__typename,
        id: o.id,
        isBookmarked: getYou(o).isBookmarked,
        label: getDisplay(o, languages).title,
        runnableObject: o.__typename === "RunProject" ?
            o.projectVersion :
            o.__typename === "RunRoutine" ?
                o.routineVersion :
                undefined,
        bookmarks: getCounts(o).bookmarks,
        to: isOfType(o, "Bookmark", "View", "Vote") ? o.to : undefined,
        user: isOfType(o, "Member") ? o.user : undefined,
        versions: isOfType(o, "Api", "Note", "Project", "Routine", "SmartContract", "Standard") ? o.versions : undefined,
        root: isOfType(o, "ApiVersion", "NoteVersion", "ProjectVersion", "RoutineVersion", "SmartContractVersion", "StandardVersion") ? o.root : undefined,
    }));
}
export function listToListItems({ beforeNavigation, dummyItems, keyPrefix, hideUpdateButton, items, loading, zIndex, }) {
    const listItems = [];
    if (loading) {
        if (!dummyItems)
            return listItems;
        for (let i = 0; i < dummyItems.length; i++) {
            listItems.push(_jsx(ObjectListItem, { data: null, hideUpdateButton: hideUpdateButton, index: i, loading: true, objectType: dummyItems[i], zIndex: zIndex }, `${keyPrefix}-${i}`));
        }
    }
    if (!items)
        return listItems;
    for (let i = 0; i < items.length; i++) {
        let curr = items[i];
        if (isOfType(curr, "Bookmark", "View", "Vote"))
            curr = curr.to;
        listItems.push(_jsx(ObjectListItem, { beforeNavigation: beforeNavigation, data: curr, hideUpdateButton: hideUpdateButton, index: i, loading: false, objectType: curr.__typename, zIndex: zIndex }, `${keyPrefix}-${curr.id}`));
    }
    return listItems;
}
const placeholderColors = [
    ["#197e2c", "#b5ffc4"],
    ["#b578b6", "#fecfea"],
    ["#4044d6", "#e1c7f3"],
    ["#d64053", "#fbb8c5"],
    ["#d69440", "#e5d295"],
    ["#40a4d6", "#79e0ef"],
    ["#6248e4", "#aac3c9"],
    ["#8ec22c", "#cfe7b4"],
];
export const placeholderColor = () => {
    return placeholderColors[Math.floor(Math.random() * placeholderColors.length)];
};
export const toSearchListData = (searchType, placeholder, where) => ({
    searchType,
    placeholder,
    where,
});
//# sourceMappingURL=listTools.js.map