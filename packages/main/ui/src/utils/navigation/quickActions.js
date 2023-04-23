import { LINKS } from "@local/consts";
import { userProfileUpdate } from "../../api/generated/endpoints/user_profileUpdate";
import { documentNodeWrapper, errorToCode } from "../../api/utils";
import { getCurrentUser } from "../authentication/session";
import { PubSub } from "../pubsub";
import { clearSearchHistory } from "../search/clearSearchHistory";
import { HistoryPageTabOption, SearchPageTabOption } from "../search/objectToSearch";
const createKeywords = ["Create", "New", "AddNew", "CreateNew"];
const searchKeywords = ["Search", "Find", "LookFor", "LookUp"];
const viewKeywords = ["View", "ViewPage", "GoTo", "Navigate"];
export const shortcuts = [
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
        value: `${LINKS.Notifications}`,
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
export const actions = [
    {
        label: "Clear search history",
        id: "clear-search-history",
        canPerform: () => true,
    },
    {
        label: "Activate dark mode",
        id: "activate-dark-mode",
        canPerform: (session) => getCurrentUser(session).theme !== "dark",
    },
    {
        label: "Activate light mode",
        id: "activate-light-mode",
        canPerform: (session) => getCurrentUser(session).theme !== "light",
    },
];
export const actionsItems = actions.map(({ canPerform, id, label }) => ({
    __typename: "Action",
    canPerform,
    id,
    label,
}));
export const performAction = async (option, session) => {
    switch (option.id) {
        case "clear-search-history":
            session && clearSearchHistory(session);
            break;
        case "activate-dark-mode":
            if (session?.isLoggedIn) {
                documentNodeWrapper({
                    node: userProfileUpdate,
                    input: { theme: "dark" },
                    onSuccess: () => { PubSub.get().publishTheme("dark"); },
                    onError: (error) => { PubSub.get().publishSnack({ messageKey: errorToCode(error), severity: "Error", data: error }); },
                });
            }
            else
                PubSub.get().publishTheme("dark");
            break;
        case "activate-light-mode":
            if (session?.isLoggedIn) {
                documentNodeWrapper({
                    node: userProfileUpdate,
                    input: { theme: "light" },
                    onSuccess: () => { PubSub.get().publishTheme("light"); },
                    onError: (error) => { PubSub.get().publishSnack({ messageKey: errorToCode(error), severity: "Error", data: error }); },
                });
            }
            else
                PubSub.get().publishTheme("light");
            break;
    }
};
//# sourceMappingURL=quickActions.js.map