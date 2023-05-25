import { LINKS, ProfileUpdateInput, Session, User } from "@local/shared";
import { userProfileUpdate } from "api/generated/endpoints/user_profileUpdate";
import { documentNodeWrapper, errorToCode } from "api/utils";
import { ActionOption } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { clearSearchHistory } from "utils/search/clearSearchHistory";
import { HistoryPageTabOption, SearchPageTabOption } from "utils/search/objectToSearch";
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
        label: "CreateApi",
        keywords: createKeywords,
        value: `${LINKS.Api}/add`,
    },
    {
        label: "CreateNote",
        keywords: createKeywords,
        value: `${LINKS.Note}/add`,
    },
    {
        label: "CreateOrganization",
        keywords: createKeywords,
        value: `${LINKS.Organization}/add`,
    },
    {
        label: "CreateProject",
        keywords: createKeywords,
        value: `${LINKS.Project}/add`,
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
        label: "CreateRoutine",
        keywords: createKeywords,
        value: `${LINKS.Routine}/add`,
    },
    {
        label: "CreateSmartContract",
        keywords: createKeywords,
        value: `${LINKS.SmartContract}/add`,
    },
    {
        label: "CreateStandard",
        keywords: createKeywords,
        value: `${LINKS.Standard}/add`,
    },
    {
        label: "ViewHistory",
        keywords: viewKeywords,
        value: `${LINKS.History}`,
    },
    {
        label: "ViewNotifications",
        keywords: viewKeywords,
        value: `${LINKS.Inbox}`,
    },
    {
        label: "ViewProfile",
        keywords: viewKeywords,
        value: `${LINKS.Profile}`,
    },
    {
        label: "ViewSettings",
        keywords: viewKeywords,
        value: `${LINKS.Settings}`,
    },
    {
        label: "SearchApi",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type=${SearchPageTabOption.Apis}`,
    },
    {
        label: "SearchNote",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type=${SearchPageTabOption.Notes}`,
    },
    {
        label: "SearchOrganization",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type=${SearchPageTabOption.Organizations}`,
    },
    {
        label: "SearchQuestion",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type=${SearchPageTabOption.Questions}`,
    },
    {
        label: "SearchProject",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type=${SearchPageTabOption.Projects}`,
    },
    {
        label: "SearchQuestion",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type=${SearchPageTabOption.Questions}`,
    },
    {
        label: "SearchRoutine",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type=${SearchPageTabOption.Routines}`,
    },
    {
        label: "SearchSmartContract",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type=${SearchPageTabOption.SmartContracts}`,
    },
    {
        label: "SearchStandard",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type=${SearchPageTabOption.Standards}`,
    },
    {
        label: "SearchUser",
        keywords: searchKeywords,
        value: `${LINKS.Search}?type=${SearchPageTabOption.Users}`,
    },
    {
        label: "SearchRun",
        keywords: searchKeywords,
        value: `${LINKS.History}?type=${HistoryPageTabOption.RunsActive}`,
    },
    {
        label: "SearchView",
        keywords: [...searchKeywords, "SearchViewed"],
        value: `${LINKS.History}?type=${HistoryPageTabOption.Viewed}`,
    },
    {
        label: "SearchBookmark",
        keywords: [...searchKeywords, "SearchBookmarked"],
        value: `${LINKS.History}?type=${HistoryPageTabOption.Bookmarked}`,
    },
    // { //TODO should be possible to replicate with normal advanced search
    //     label: 'Search your actively developing projects and routines',
    //     link: `${LINKS.DevelopSearch}?type=${DevelopSearchPageTabOption.InProgress}`,
    // },
    // {
    //     label: 'Search your completed projects and routines',
    //     link: `${LINKS.DevelopSearch}?type=${DevelopSearchPageTabOption.Completed}`,
    // },
    // {
    //     label: 'Search your actively developing projects and routines advanced',
    //     link: `${LINKS.DevelopSearch}?type=${DevelopSearchPageTabOption.InProgress}&advanced=true`,
    // },
    // {
    //     label: 'Search your completed projects and routines advanced',
    //     link: `${LINKS.DevelopSearch}?type=${DevelopSearchPageTabOption.Completed}&advanced=true`,
    // },
    {
        label: "BeginnersGuide",
        keywords: viewKeywords,
        value: `${LINKS.Welcome}`,
    },
    {
        label: "Faq",
        keywords: viewKeywords,
        value: `${LINKS.FAQ}`,
    },
    {
        label: "Tutorial",
        keywords: viewKeywords,
        value: `${LINKS.Tutorial}`,
    },
];

/**
 * Action shortcuts that can appear in the main search bar or command palette. 
 * Instead of taking you to a page, they perform an action (e.g. clear search history).
 */
export const actions: ActionItem[] = [
    {
        label: "Clear search history",
        id: "clear-search-history",
        canPerform: () => true,
    },
    {
        label: "Activate dark mode",
        id: "activate-dark-mode",
        canPerform: (session: Session) => getCurrentUser(session).theme !== "dark",
    },
    {
        label: "Activate light mode",
        id: "activate-light-mode",
        canPerform: (session: Session) => getCurrentUser(session).theme !== "light",
    },
];

/**
 * Shape actions to match AutoCompleteListItem format.
 */
export const actionsItems: ActionOption[] = actions.map(({ canPerform, id, label }) => ({
    __typename: "Action",
    canPerform,
    id,
    label,
}));

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
                documentNodeWrapper<User, ProfileUpdateInput>({
                    node: userProfileUpdate,
                    input: { theme: "dark" },
                    onSuccess: () => { PubSub.get().publishTheme("dark"); },
                    onError: (error) => { PubSub.get().publishSnack({ messageKey: errorToCode(error), severity: "Error", data: error }); },
                });
            }
            // Otherwise, just publish theme change.
            else PubSub.get().publishTheme("dark");
            break;
        case "activate-light-mode":
            // If logged in, update user profile and publish theme change.
            if (session?.isLoggedIn) {
                documentNodeWrapper<User, ProfileUpdateInput>({
                    node: userProfileUpdate,
                    input: { theme: "light" },
                    onSuccess: () => { PubSub.get().publishTheme("light"); },
                    onError: (error) => { PubSub.get().publishSnack({ messageKey: errorToCode(error), severity: "Error", data: error }); },
                });
            }
            // Otherwise, just publish theme change.
            else PubSub.get().publishTheme("light");
            break;
    }
};
