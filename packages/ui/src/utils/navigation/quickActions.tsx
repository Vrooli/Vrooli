import { ActionOption, endpointsUser, HistoryPageTabOption, LINKS, PreActionOption, ProfileUpdateInput, SearchPageTabOption, Session, User } from "@local/shared";
import { fetchWrapper } from "../../api/fetchWrapper.js";
import { ActionIcon, ApiIcon, BookmarkFilledIcon, HelpIcon, NoteIcon, PlayIcon, ProjectIcon, RoutineIcon, ShortcutIcon, StandardIcon, TeamIcon, TerminalIcon, UserIcon, VisibleIcon } from "../../icons/common.js";
import { SvgComponent } from "../../types.js";
import { getCurrentUser } from "../authentication/session.js";
import { PubSub } from "../pubsub.js";
import { SearchHistory } from "../search/searchHistory.js";
import { PreSearchItem } from "../search/siteToSearch.js";

export interface ShortcutItem {
    label: string;
    link: string;
}

const createKeywords = ["Create", "New", "AddNew", "CreateNew"] as const;
const searchKeywords = ["Search", "Find", "LookFor", "LookUp"] as const;
const viewKeywords = ["View", "ViewPage", "GoTo", "Navigate"] as const;

export const shortcuts: PreSearchItem[] = [
    {
        label: "History",
        keywords: viewKeywords,
        value: `${LINKS.History}`,
    },
    {
        label: "Inbox",
        keywords: viewKeywords,
        value: `${LINKS.Inbox}`,
    },
    {
        label: "MyStuff",
        value: LINKS.MyStuff,
    },
    {
        label: "Profile",
        keywords: viewKeywords,
        value: `${LINKS.Profile}`,
    },
    {
        label: "Settings",
        keywords: viewKeywords,
        value: `${LINKS.Settings}`,
    },
    {
        label: "Create",
        keywords: createKeywords,
        value: LINKS.Create,
    },
    {
        label: "CreateApi",
        keywords: createKeywords,
        value: `${LINKS.Api}/add`,
    },
    {
        label: "CreateBot",
        keywords: createKeywords,
        value: `${LINKS.User}/add`,
    },
    {
        label: "CreateChat",
        keywords: createKeywords,
        value: `${LINKS.Chat}/add`,
    },
    {
        label: "CreateDataConverter",
        keywords: createKeywords,
        value: `${LINKS.DataConverter}/add`,
    },
    {
        label: "CreateDataStructure",
        keywords: createKeywords,
        value: `${LINKS.DataStructure}/add`,
    },
    {
        label: "CreateNote",
        keywords: createKeywords,
        value: `${LINKS.Note}/add`,
    },
    {
        label: "CreateRoutineMultiStep",
        keywords: createKeywords,
        value: `${LINKS.RoutineMultiStep}/add`,
    },
    {
        label: "CreateRoutineSingleStep",
        keywords: createKeywords,
        value: `${LINKS.RoutineSingleStep}/add`,
    },
    {
        label: "CreateProject",
        keywords: createKeywords,
        value: `${LINKS.Project}/add`,
    },
    {
        label: "CreatePrompt",
        keywords: createKeywords,
        value: `${LINKS.Prompt}/add`,
    },
    {
        label: "CreateQuestion",
        keywords: createKeywords,
        value: `${LINKS.Question}/add`,
    },
    {
        label: "CreateReminder",
        keywords: createKeywords,
        value: `${LINKS.Reminder}/add`,
    },
    {
        label: "CreateSmartContract",
        keywords: createKeywords,
        value: `${LINKS.SmartContract}/add`,
    },
    {
        label: "CreateTeam",
        keywords: createKeywords,
        value: `${LINKS.Team}/add`,
    },
    {
        label: "Search",
        keywords: searchKeywords,
        value: LINKS.Search,
    },
    {
        label: "SearchApi",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type="${SearchPageTabOption.Api}"`,
    },
    {
        label: "SearchDataConverter",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type="${SearchPageTabOption.DataConverter}"`,
    },
    {
        label: "SearchDataStructure",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type="${SearchPageTabOption.DataStructure}"`,
    },
    {
        label: "SearchProject",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type="${SearchPageTabOption.Project}"`,
    },
    {
        label: "SearchPrompt",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type="${SearchPageTabOption.Prompt}"`,
    },
    {
        label: "SearchNote",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type="${SearchPageTabOption.Note}"`,
    },
    {
        label: "SearchQuestion",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type="${SearchPageTabOption.Question}"`,
    },
    {
        label: "SearchRoutineMultiStep",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type="${SearchPageTabOption.RoutineMultiStep}"`,
    },
    {
        label: "SearchRoutineSingleStep",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type="${SearchPageTabOption.RoutineSingleStep}"`,
    },
    {
        label: "SearchRun",
        keywords: searchKeywords,
        value: `${LINKS.History}?type="${HistoryPageTabOption.RunsActive}"`,
    },
    {
        label: "SearchSmartContract",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type="${SearchPageTabOption.SmartContract}"`,
    },
    {
        label: "SearchTeam",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type="${SearchPageTabOption.Team}"`,
    },
    {
        label: "SearchUser",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type="${SearchPageTabOption.User}"`,
    },
    {
        label: "SearchView",
        keywords: [...searchKeywords, "SearchViewed"],
        value: `${LINKS.History}?type="${HistoryPageTabOption.Viewed}"`,
    },
    {
        label: "SearchBookmark",
        keywords: [...searchKeywords, "SearchBookmarked"],
        value: `${LINKS.History}?type="${HistoryPageTabOption.Bookmarked}"`,
    },
    // { //TODO should be possible to replicate with normal advanced search
    //     label: 'Search your actively developing projects and routines',
    //     link: `${LINKS.DevelopSearch}?type="${DevelopSearchPageTabOption.InProgress}"`,
    // },
    // {
    //     label: 'Search your completed projects and routines',
    //     link: `${LINKS.DevelopSearch}?type="${DevelopSearchPageTabOption.Completed}"`,
    // },
    // {
    //     label: 'Search your actively developing projects and routines advanced',
    //     link: `${LINKS.DevelopSearch}?type="${DevelopSearchPageTabOption.InProgress}"&advanced=true`,
    // },
    // {
    //     label: 'Search your completed projects and routines advanced',
    //     link: `${LINKS.DevelopSearch}?type="${DevelopSearchPageTabOption.Completed}"&advanced=true`,
    // },
];

/**
 * Action shortcuts that can appear in the main search bar or command palette. 
 * Instead of taking you to a page, they perform an action (e.g. clear search history).
 */
export const Actions: { [x: string]: PreActionOption } = {
    clearSearchHistory: {
        __typename: "Action",
        label: "ClearSearchHistory",
        id: "clear-search-history",
        canPerform: () => true,
        keywords: ["Clear", "Delete", "Remove", "Erase"] as const,
    },
    activateDarkMode: {
        __typename: "Action",
        label: "ActivateDarkMode",
        id: "activate-dark-mode",
        canPerform: (session: Session) => getCurrentUser(session).theme !== "dark",
        keywords: ["Dark", "Black", "Theme"] as const,
    },
    activateLightMode: {
        __typename: "Action",
        label: "ActivateLightMode",
        id: "activate-light-mode",
        canPerform: (session: Session) => getCurrentUser(session).theme !== "light",
        keywords: ["Light", "White", "Theme"] as const,
    },
    tutorial: {
        __typename: "Action",
        label: "Tutorial",
        id: "tutorial",
        canPerform: (session: Session) => session.isLoggedIn,
        keywords: ["Tutorial", "Help", "Learn"] as const,
    },
};

export const actionsItems: PreActionOption[] = Object.values(Actions);

/**
 * Maps action ids to their corresponding action. 
 * Actions cannot be stored in the options themselves because localStorage cannot store functions.
 */
export async function performAction(option: PreActionOption | ActionOption, session: Session | null | undefined): Promise<void> {
    switch (option.id) {
        case "clear-search-history":
            session && SearchHistory.clearSearchHistory(session);
            break;
        case "activate-dark-mode":
            // If logged in, update user profile and publish theme change.
            if (session?.isLoggedIn) {
                fetchWrapper<ProfileUpdateInput, User>({
                    ...endpointsUser.profileUpdate,
                    inputs: { theme: "dark" },
                    onSuccess: () => { PubSub.get().publish("theme", "dark"); },
                });
            }
            // Otherwise, just publish theme change.
            else PubSub.get().publish("theme", "dark");
            break;
        case "activate-light-mode":
            // If logged in, update user profile and publish theme change.
            if (session?.isLoggedIn) {
                fetchWrapper<ProfileUpdateInput, User>({
                    ...endpointsUser.profileUpdate,
                    inputs: { theme: "light" },
                    onSuccess: () => { PubSub.get().publish("theme", "light"); },
                });
            }
            // Otherwise, just publish theme change.
            else PubSub.get().publish("theme", "light");
            break;
        case "tutorial":
            if (session?.isLoggedIn) {
                PubSub.get().publish("menu", { id: ELEMENT_IDS.Tutorial, isOpen: true });
            } else {
                PubSub.get().publish("snack", { messageKey: "NotLoggedIn", severity: "Error" });
            }
            break;
    }
}

const IconMap = {
    Action: ActionIcon,
    Api: ApiIcon,
    Bookmark: BookmarkFilledIcon,
    Code: TerminalIcon,
    Note: NoteIcon,
    Project: ProjectIcon,
    Question: HelpIcon,
    Routine: RoutineIcon,
    Run: PlayIcon,
    Shortcut: ShortcutIcon,
    Standard: StandardIcon,
    Team: TeamIcon,
    User: UserIcon,
    View: VisibleIcon,
};

/**
 * Maps object types to icons
 */
export function getAutocompleteOptionIcon(type: string, fill: string): JSX.Element | null {
    const Icon: SvgComponent | undefined = IconMap[type];
    return Icon ? <Icon fill={fill} /> : null;
}
