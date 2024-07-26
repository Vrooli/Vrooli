import { Api, ApiVersion, AutocompleteOption, Bookmark, BookmarkFor, Chat, ChatInvite, ChatParticipant, Code, CodeVersion, CommentFor, CopyType, DUMMY_ID, DeleteType, DotNotation, ListObject, Meeting, Member, MemberInvite, NodeRoutineListItem, Note, NoteVersion, Project, ProjectVersion, Reaction, ReactionFor, ReportFor, Resource, ResourceList, Routine, RoutineVersion, RunProject, RunRoutine, Standard, StandardVersion, User, View, YouInflated, exists, getTranslation, isOfType } from "@local/shared";
import { Chip, Palette } from "@mui/material";
import { BotIcon } from "icons";
import { routineTypes } from "utils/search/schemas/routine";
import { valueFromDot } from "utils/shape/general";
import { displayDate, firstString } from "./stringTools";
import { getUserLanguages } from "./translationTools";

/**
 * Most possible counts (including score) any object can have
 */
export type CountsInflated = {
    comments: number;
    forks: number;
    issues: number;
    labels: number;
    pullRequests: number;
    questions: number;
    reports: number;
    score: number;
    bookmarks: number;
    transfers: number;
    translations: number;
    versions: number;
    views: number;
}

/**
 * Finds dot notation for the location of the "you" property in an object which contains the specified property
 * @param object An object
 * @param property A property to find in the "you" property of the object
 */
export function getYouDot(
    object: ListObject | null | undefined,
    property: keyof YouInflated,
): DotNotation<typeof object> | null {
    // If no object, return null
    if (!object) return null;
    // If the object is a bookmark, reaction, or view, use the "to" object
    if (isOfType(object, "Bookmark", "Reaction", "View")) return getYouDot((object as Partial<Bookmark | Reaction | View>).to, property);
    // If the object is a run routine, use the routine version
    if (isOfType(object, "RunRoutine")) return getYouDot((object as Partial<RunRoutine>).routineVersion, property);
    // If the object is a run project, use the project version
    if (isOfType(object, "RunProject")) return getYouDot((object as Partial<RunProject>).projectVersion, property);
    // Check object.you
    if (exists(object.you?.[property])) return `you.${property}`;
    // Check object.root.you
    if (exists(object.root?.you?.[property])) return `root.you.${property}`;
    // If not found, return null
    return null;
}

export const defaultYou: YouInflated = {
    canComment: false,
    canCopy: false,
    canDelete: false,
    canRead: false,
    canReport: false,
    canShare: true,
    canBookmark: false,
    canUpdate: false,
    canReact: false,
    isBookmarked: false,
    isViewed: false,
    reaction: null,
};

/**
 * Gets user permissions and statuses for an object. These are inflated to match YouInflated, so any fields not present are false
 * @param object An object
 */
export function getYou(
    object: ListObject | null | undefined,
): YouInflated {
    // Initialize fields to false (except reaction, since that's an emoji or null instead of a boolean)
    const objectPermissions = { ...defaultYou };
    if (!object) return objectPermissions;
    // Shortcut: If ID is DUMMY_ID, then it's a new object and you only delete it
    if (object.id === DUMMY_ID) return {
        ...Object.keys(defaultYou).reduce((acc, key) => ({ ...acc, [key]: typeof defaultYou[key] === "boolean" ? false : defaultYou[key] }), {}),
        canDelete: true,
    } as YouInflated;
    // Helper function to get permissions
    function getPermission(key: keyof YouInflated): boolean {
        // Check if the field is in the object
        const field = valueFromDot(object, `you.${key}`);
        if (field === true || field === false || typeof field === "boolean") return field;
        // If not, check if the field is in the root.you object
        const rootField = valueFromDot(object, `root.you.${key}`);
        if (rootField === true || rootField === false || typeof rootField === "boolean") return rootField;
        return false; // Default to false if no field found
    }
    // Some permissions are based on a relation (e.g. bookmarking a View's "to" relation), 
    // while others are always based on the current object (e.g. deleting a member instead of the user it's associated with).
    // Keep this in mind when looking at the code below.
    if (isOfType(object, "Bookmark", "Reaction", "View")) return {
        ...getYou((object as Partial<Bookmark | Reaction | View>).to),
        canDelete: getPermission("canDelete"),
    };
    if (isOfType(object, "RunRoutine")) return {
        ...getYou((object as Partial<RunRoutine>).routineVersion),
        canBookmark: false,
        canComment: false,
        canReact: false,
        canShare: false,
        canDelete: getPermission("canDelete"),
    };
    if (isOfType(object, "RunProject")) return {
        ...getYou((object as Partial<RunProject>).projectVersion),
        canBookmark: false,
        canComment: false,
        canReact: false,
        canShare: false,
        canDelete: getPermission("canDelete"),
    };
    if (isOfType(object, "Member", "MemberInvite", "ChatParticipant", "ChatInvite")) return {
        ...getYou((object as Partial<Member | MemberInvite | ChatParticipant | ChatInvite>).user),
        canCopy: getPermission("canCopy"),
        canDelete: getPermission("canDelete"),
        canUpdate: getPermission("canUpdate"),
    };
    if (isOfType(object, "Resource")) return getYou((object as Partial<Resource>).list);
    if (isOfType(object, "ResourceList")) return getYou((object as Partial<ResourceList>).listFor);
    // Loop through all permission fields
    for (const key in objectPermissions) {
        objectPermissions[key] = getPermission(key as keyof YouInflated);
    }
    // Now remove permissions is the action is not allowed on the object type (e.g. can't react to a user).
    function filterInvalidAction(action: keyof YouInflated, enumType: Record<string, unknown>) {
        if (!object) return;
        if (objectPermissions[action] && [object.__typename, object.__typename + "Version", object.__typename.replace("Version", "")].every(type => !exists(enumType[type]))) {
            objectPermissions[action as any] = false;
        }
    }
    filterInvalidAction("canBookmark", BookmarkFor);
    filterInvalidAction("canComment", CommentFor);
    filterInvalidAction("canCopy", CopyType);
    filterInvalidAction("canDelete", DeleteType);
    filterInvalidAction("canReact", ReactionFor);
    filterInvalidAction("canReport", ReportFor);
    return objectPermissions;
}

/**
 * Gets counts for an object. These are inflated to match CountsInflated, so any fields not present are 0
 * @param object An object
 */
export function getCounts(
    object: ListObject | null | undefined,
): CountsInflated {
    // Initialize fields to 0
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
    if (!object) return defaultCounts;
    // If a bookmark, reaction, or view, use the "to" object
    if (isOfType(object, "Bookmark", "Reaction", "View")) return getCounts((object as Partial<Bookmark | Reaction | View>).to);
    // If a run routine, use the routine version
    if (isOfType(object, "RunRoutine")) return getCounts((object as Partial<RunRoutine>).routineVersion);
    // If a run project, use the project version
    if (isOfType(object, "RunProject")) return getCounts((object as Partial<RunProject>).projectVersion);
    // If a NodeRoutineListItem, use the routine version
    if (isOfType(object, "NodeRoutineListItem")) return getCounts((object as Partial<NodeRoutineListItem>).routineVersion);
    // Otherwise, get the counts from the object
    // Loop through all count fields
    for (const key in defaultCounts) {
        // For every field except score and views, property name is field + "Count"
        const objectProp = ["score", "views"].includes(key) ? key : `${key}Count`;
        // Check if the field is in the object
        const field = valueFromDot(object, objectProp);
        if (field !== undefined) defaultCounts[key] = field;
        // If not, check if the field is in the root.counts object
        else {
            const field = valueFromDot(object, `root.${objectProp}`);
            if (field !== undefined) defaultCounts[key] = field;
        }
    }
    return defaultCounts;
}

/**
 * Attempts to find the most relevant title for an object. Does not check root object or versions
 * @param obj An object with (hopefully) a title
 * @param langs The user's preferred languages
 * @returns The title, or null if none found
 */
function tryTitle(obj: Record<string, unknown>, langs: readonly string[]) {
    const translations: Record<string, unknown> = getTranslation(obj, langs, true);
    // The order of these is important to display the most relevant title
    return firstString(
        obj.title,
        obj.name,
        obj.label,
        translations.title,
        translations.name,
        translations.label,
        obj.handle ? `$${obj.handle}` : null,
    );
}

/**
 * Attempts to find the most relevant subtitle for an object. Does not check root object or versions
 * @param obj An object with (hopefully) a subtitle
 * @param langs The user's preferred languages
 * @returns The subtitle, or null if none found
 */
function trySubtitle(obj: Record<string, unknown>, langs: readonly string[]) {
    const translations: Record<string, unknown> = getTranslation(obj, langs, true);
    return firstString(
        obj.bio,
        obj.description,
        obj.summary,
        obj.details,
        obj.text,
        translations.bio,
        translations.description,
        translations.summary,
        translations.details,
        translations.text,
    );
}

/**
 * For an object which does not have a direct title (i.e. it's likely in the root object or a version), 
 * tries to find the most relevant title and subtitle
 * @param obj An object
 * @param langs The user's preferred languages
 * @returns The title and subtitle, or blank strings if none found
 */
function tryVersioned(obj: Record<string, any>, langs: readonly string[]) {
    // Initialize the title and subtitle
    let title: string | null = null;
    let subtitle: string | null = null;
    // Create a list of objects to check. Order is important
    const objectsToCheck = [
        obj, // The object itself
        obj.root, // The root object (only found if obj is a version)
        obj.versions?.find(v => v.isLatest), // The latest version (only found if obj is a root object)
        ...([...(obj.versions ?? [])].sort((a, b) => b.versionIndex - a.versionIndex)), // All versions, sorted by versionIndex (i.e. newest first)
    ];
    // Loop through the objects
    for (const curr of objectsToCheck) {
        // If the object is null or undefined, skip it
        if (!exists(curr)) continue;
        // Check for a title if we haven't found one yet
        if (!title) {
            title = tryTitle(curr, langs);
        }
        // Check for a subtitle if we haven't found one yet
        if (!subtitle) {
            subtitle = trySubtitle(curr, langs);
        }
        // If both are found, break
        if (title && subtitle) break;
    }
    return { title: title ?? "", subtitle: subtitle ?? "" };
}

export type DisplayAdornment = {
    Adornment: JSX.Element,
    key: string,
};

type GetDisplayResult = {
    title: string,
    subtitle: string,
    adornments: DisplayAdornment[],
};

/**
 * Gets the name and subtitle of a list object
 * @param object A list object
 * @param languages User languages
 * @returns The name and subtitle of the object
 */
export function getDisplay(
    object: ListObject | null | undefined,
    languages?: readonly string[],
    palette?: Palette,
): GetDisplayResult {
    const adornments: GetDisplayResult["adornments"] = [];
    if (!object) return { title: "", subtitle: "", adornments };
    // If a bookmark, reaction, or view, use the "to" object
    if (isOfType(object, "Bookmark", "Reaction", "View")) return getDisplay((object as Partial<Bookmark | Reaction | View>).to as ListObject);
    const langs: readonly string[] = languages ?? getUserLanguages(undefined);
    // If a run routine, use the routine version's display and the startedAt/completedAt date
    if (isOfType(object, "RunRoutine")) {
        const { completedAt, name, routineVersion, startedAt } = object as Partial<RunRoutine>;
        const title = firstString(name, getTranslation(routineVersion, langs, true).name);
        const started = startedAt ? displayDate(startedAt) : null;
        const completed = completedAt ? displayDate(completedAt) : null;
        const { subtitle: routineVersionSubtitle } = getDisplay(routineVersion, langs);
        return {
            title,
            subtitle: (started ? "Started: " + started : completed ? "Completed: " + completed : "") + (routineVersionSubtitle ? " | " + routineVersionSubtitle : ""),
            adornments,
        };
    }
    // If a run project, use the project version's display and the startedAt/completedAt date
    if (isOfType(object, "RunProject")) {
        const { completedAt, name, projectVersion, startedAt } = object as Partial<RunProject>;
        const title = firstString(name, getTranslation(projectVersion, langs, true).name);
        const started = startedAt ? displayDate(startedAt) : null;
        const completed = completedAt ? displayDate(completedAt) : null;
        const { subtitle: projectVersionSubtitle } = getDisplay(projectVersion, langs);
        return {
            title,
            subtitle: (started ? "Started: " + started : completed ? "Completed: " + completed : "") + (projectVersionSubtitle ? " | " + projectVersionSubtitle : ""),
            adornments,
        };
    }
    // If a chat or meeting, use it's title/subtitle, or default to descriptive text with participant or participant count
    if (isOfType(object, "Chat", "Meeting")) {
        const participants = (object as Partial<Meeting>).attendees ?? (object as Partial<Chat>).participants ?? [];
        const participantsCount = (object as Partial<Meeting>).attendeesCount ?? (object as Partial<Chat>).participantsCount;
        const updated_at = (object as Partial<Chat | Meeting>).updated_at;
        const { name, description } = getTranslation(object as Partial<Chat>, langs, true);
        const isGroup = Number.isInteger(participantsCount) && (participantsCount as number) > 2;
        const firstUser = (participants as unknown as Meeting["attendees"])[0] ?? (participants as unknown as Chat["participants"])[0]?.user;
        const title = firstString(name, isGroup ? //TODO internationalize this and support meeting
            `Group chat (${participantsCount})` :
            firstUser ?
                `Chat with ${getDisplay(firstUser).title}` :
                "Chat",
        );
        const subtitle = firstString(description, displayDate(updated_at));
        return { title, subtitle, adornments };
    }
    // If a member or chat participant, use the user's display
    if (isOfType(object, "Member", "MemberInvite", "ChatParticipant", "ChatInvite")) return getDisplay({ __typename: "User", ...(object as Partial<ChatParticipant | ChatInvite | Member | MemberInvite>).user } as ListObject);
    // For all other objects, fields may differ. 
    const { title, subtitle } = tryVersioned(object, langs);
    // If a NodeRoutineListItem, use the routine version's display if title or subtitle is empty
    if (isOfType(object, "NodeRoutineListItem") && title.length === 0 && subtitle.length === 0) {
        const routineVersionDisplay = getDisplay({ __typename: "RoutineVersion", ...(object as Partial<NodeRoutineListItem>).routineVersion } as ListObject, langs);
        return {
            title: firstString(title, routineVersionDisplay.title),
            subtitle: firstString(subtitle, routineVersionDisplay.subtitle),
            adornments: [],
        };
    }
    // If a User, and `isBot` is true, add BotIcon to adornments
    if (isOfType(object, "User") && (object as Partial<User>).isBot) {
        adornments.push({
            Adornment: <BotIcon
                fill={palette?.mode === "light" ? "#521f81" : "#a979d5"}
                width="100%"
                height="100%"
                style={{ padding: "1px" }}
            />,
            key: "bot",
        });
        // If the bot is depicting a real person, add a chip indicating that
        if ((object as Partial<User>).isBotDepictingPerson) {
            adornments.push({
                Adornment: <Chip key="parody" label="Parody" sx={{ backgroundColor: palette?.mode === "light" ? "#521f81" : "#a979d5", color: "white", display: "inline" }} />,
                key: "parody",
            });
        }
    }
    // If a Routine, add adornment for routine type
    if (isOfType(object, "Routine", "RoutineVersion")) {
        const routineType = (object as Partial<RoutineVersion>).routineType ?? (object as Partial<Routine>).versions?.find(v => v.isLatest)?.routineType;
        const routineTypeInfo = routineType ? routineTypes.find(t => t.type === routineType) : undefined;
        if (routineTypeInfo) {
            adornments.push({
                Adornment: <Chip key="routine-type" label={routineTypeInfo.label} sx={{ backgroundColor: "#001b76", color: "white", display: "inline" }} />,
                key: `routine-type-chip-${routineTypeInfo.type}`,
            });
        }
    }
    // Return result
    return { title, subtitle, adornments };
}

/**
 * Finds the information required to bookmark an object
 * @param object 
 * @returns BookmarkFor type and ID of the object. For versions, for example, 
 * the ID is of the root object instead of the version passed in.
 */
export function getBookmarkFor(
    object: ListObject | null | undefined,
): { bookmarkFor: BookmarkFor, starForId: string } | { bookmarkFor: null, starForId: null } {
    if (!object || !object.id) return { bookmarkFor: null, starForId: null };
    // If object does not support bookmarking, return null
    if (isOfType(object, "BookmarkList", "Member", "MemberInvite", "ChatParticipant", "ChatInvite")) return { bookmarkFor: null, starForId: null }; //TODO add more types
    // If a bookmark, reaction, or view, use the "to" object
    if (isOfType(object, "Bookmark", "Reaction", "View")) return getBookmarkFor((object as Partial<Bookmark | Reaction | View>).to);
    // If a run routine, use the routine version
    if (isOfType(object, "RunRoutine")) return getBookmarkFor((object as Partial<RunRoutine>).routineVersion);
    // If a run project, use the project version
    if (isOfType(object, "RunProject")) return getBookmarkFor((object as Partial<RunProject>).projectVersion);
    // If a NodeRoutineListItem, use the routine version
    if (isOfType(object, "NodeRoutineListItem")) return getBookmarkFor((object as Partial<NodeRoutineListItem>).routineVersion);
    // If the object contains a root object, use that
    if (Object.prototype.hasOwnProperty.call(object, "root"))
        return getBookmarkFor((object as Partial<ApiVersion | CodeVersion | NoteVersion | ProjectVersion | RoutineVersion | StandardVersion>).root);
    // Use current object
    return { bookmarkFor: object.__typename as unknown as BookmarkFor, starForId: object.id };
}

/**
 * Converts a list of GraphQL objects to a list of autocomplete information.
 * @param objects The list of search results
 * @param languages User languages
 * @returns The list of autocomplete information
 */
export function listToAutocomplete(
    objects: readonly ListObject[],
    languages: readonly string[],
): AutocompleteOption[] {
    return objects.map(o => ({
        __typename: o.__typename,
        id: o.id,
        isBookmarked: getYou(o).isBookmarked,
        key: o.id,
        label: getDisplay(o, languages).title,
        runnableObject: o.__typename === "RunProject" ?
            o.projectVersion :
            o.__typename === "RunRoutine" ?
                o.routineVersion :
                undefined,
        bookmarks: getCounts(o).bookmarks,
        to: isOfType(o, "Bookmark", "Reaction", "View") ?
            (o as Partial<Bookmark | Reaction | View>).to :
            undefined,
        user: isOfType(o, "Member", "MemberInvite", "ChatParticipant", "ChatInvite") ?
            (o as Partial<Member | MemberInvite | ChatParticipant | ChatInvite>).user :
            undefined,
        versions: isOfType(o, "Api", "Note", "Project", "Routine", "Code", "Standard") ?
            (o as Partial<Api | Note | Project | Routine | Code | Standard>).versions :
            undefined,
        root: isOfType(o, "ApiVersion", "CodeVersion", "NoteVersion", "ProjectVersion", "RoutineVersion", "StandardVersion") ?
            (o as Partial<ApiVersion | CodeVersion | NoteVersion | ProjectVersion | RoutineVersion | StandardVersion>).root :
            undefined,
    })) as AutocompleteOption[];
}

/**
 * Color options for placeholder icon
 * [background color, silhouette color]
 */
export const placeholderColors: [string, string][] = [
    ["#197e2c", "#b5ffc4"],
    ["#b578b6", "#fecfea"],
    ["#4044d6", "#e1c7f3"],
    ["#d64053", "#fbb8c5"],
    ["#d69440", "#e5d295"],
    ["#40a4d6", "#79e0ef"],
    ["#6248e4", "#aac3c9"],
    ["#8ec22c", "#cfe7b4"],
];

/**
 * Simple hash function to convert a string to a number.
 */
export function simpleHash(str: string): number {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
        const char = str.charCodeAt(i);
        hash = (hash << 5) - hash + char;
        hash |= 0; // Convert to 32bit integer
    }
    return hash;
};

/**
 * Finds a random color for a placeholder icon
 * @param seed Optional seed for random number generation
 * @returns [background color code, silhouette color code]
 */
export function placeholderColor(seed?: unknown): [string, string] {
    let random = Math.random();
    if (typeof seed === 'string' || typeof seed === 'number') {
        const seedStr = seed.toString();
        const hash = simpleHash(seedStr);
        random = (Math.sin(hash) + 1) / 2; // Generate a pseudo-random number from [-1, 1] to [0, 1]
    }
    return placeholderColors[Math.floor(random * placeholderColors.length)];
};
