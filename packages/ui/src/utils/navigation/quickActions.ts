import { ActionOption, endpointPutProfile, HistoryPageTabOption, LINKS, ProfileUpdateInput, SearchPageTabOption, Session, User } from "@local/shared";
import { errorToMessage, fetchWrapper } from "api";
import { getCurrentUser } from "utils/authentication/session";
import { getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { clearSearchHistory } from "utils/search/clearSearchHistory";
import { PreSearchItem } from "utils/search/siteToSearch";

export interface ShortcutItem {
    label: string;
    link: string;
}

export interface ActionItem {
    canPerform: (session: Session) => boolean;
    id: string;
    label: string;
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
        label: "CreateRoutine",
        keywords: createKeywords,
        value: `${LINKS.Routine}/add`,
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
        label: "SearchRoutine",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type="${SearchPageTabOption.Routine}"`,
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
export const Actions: { [x: string]: ActionItem } = {
    clearSearchHistory: {
        label: "Clear search history",
        id: "clear-search-history",
        canPerform: () => true,
    },
    activateDarkMode: {
        label: "Activate dark mode",
        id: "activate-dark-mode",
        canPerform: (session: Session) => getCurrentUser(session).theme !== "dark",
    },
    activateLightMode: {
        label: "Activate light mode",
        id: "activate-light-mode",
        canPerform: (session: Session) => getCurrentUser(session).theme !== "light",
    },
    tutorial: {
        label: "Tutorial",
        id: "tutorial",
        canPerform: (session: Session) => session.isLoggedIn,
    },
};

/**
 * Shape actions to match AutoCompleteListItem format.
 */
export const toActionOption = (action: ActionItem): ActionOption => ({
    __typename: "Action",
    ...action,
});
export const actionsItems: ActionOption[] = Object.values(Actions).map(toActionOption);

/**
 * Maps action ids to their corresponding action. 
 * Actions cannot be stored in the options themselves because localStorage cannot store functions.
 */
export const performAction = async (option: ActionOption, session: Session | null | undefined): Promise<void> => {
    switch (option.id) {
        case "clear-search-history":
            session && clearSearchHistory(session);
            break;
        case "activate-dark-mode":
            // If logged in, update user profile and publish theme change.
            if (session?.isLoggedIn) {
                fetchWrapper<ProfileUpdateInput, User>({
                    ...endpointPutProfile,
                    inputs: { theme: "dark" },
                    onSuccess: () => { PubSub.get().publish("theme", "dark"); },
                    onError: (error) => { PubSub.get().publish("snack", { message: errorToMessage(error, getUserLanguages(session)), severity: "Error", data: error }); },
                });
            }
            // Otherwise, just publish theme change.
            else PubSub.get().publish("theme", "dark");
            break;
        case "activate-light-mode":
            // If logged in, update user profile and publish theme change.
            if (session?.isLoggedIn) {
                fetchWrapper<ProfileUpdateInput, User>({
                    ...endpointPutProfile,
                    inputs: { theme: "light" },
                    onSuccess: () => { PubSub.get().publish("theme", "light"); },
                    onError: (error) => { PubSub.get().publish("snack", { message: errorToMessage(error, getUserLanguages(session)), severity: "Error", data: error }); },
                });
            }
            // Otherwise, just publish theme change.
            else PubSub.get().publish("theme", "light");
            break;
        case "tutorial":
            if (session?.isLoggedIn) {
                PubSub.get().publish("tutorial");
            } else {
                PubSub.get().publish("snack", { messageKey: "NotLoggedIn", severity: "Error" });
            }
            break;
    }
};
