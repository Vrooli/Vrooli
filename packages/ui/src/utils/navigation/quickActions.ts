import { APP_LINKS, ProfileUpdateInput, Session, User } from "@shared/consts";
import { SnackSeverity } from "components";
import { documentNodeWrapper, errorToCode } from "api/utils";
import { ActionOption, ShortcutOption } from "types";
import { getCurrentUser } from "utils/authentication";
import { PubSub } from "utils/pubsub";
import { userProfileUpdate } from "api/generated/endpoints/user";
import { clearSearchHistory, HistorySearchPageTabOption, PreSearchItem, SearchPageTabOption } from "utils/search";

export interface ShortcutItem {
    label: string;
    link: string;
}

export interface ActionItem {
    canPerform: (session: Session) => boolean;
    id: string;
    label: string;
}

const createKeywords = ['Create', 'New', 'AddNew', 'CreateNew'] as const;
const searchKeywords = ['Search', 'Find', 'LookFor', 'LookUp'] as const;
const searchAdvancedKeywords = ['AdvancedSearch', ...searchKeywords] as const;
const viewKeywords = ['View', 'ViewPage', 'GoTo', 'Navigate'] as const;

// TODO will replace shortcuts
export const newShortcuts: PreSearchItem[] = [
    {
        label: 'CreateApi',
        keywords: createKeywords,
        value: `${APP_LINKS.Api}/add`,
    },
    {
        label: 'CreateNote',
        keywords: createKeywords,
        value: `${APP_LINKS.Note}/add`,
    },
    {
        label: 'CreateOrganization',
        keywords: createKeywords,
        value: `${APP_LINKS.Organization}/add`,
    },
    {
        label: 'CreateProject',
        keywords: createKeywords,
        value: `${APP_LINKS.Project}/add`,
    },
    {
        label: 'CreateQuestion',
        keywords: createKeywords,
        value: `${APP_LINKS.Question}/add`,
    },
    {
        label: 'CreateReminder',
        keywords: createKeywords,
        value: `${APP_LINKS.Reminder}/add`,
    },
    {
        label: 'CreateRoutine',
        keywords: createKeywords,
        value: `${APP_LINKS.Routine}/add`,
    },
    {
        label: 'CreateSmartContract',
        keywords: createKeywords,
        value: `${APP_LINKS.SmartContract}/add`,
    },
    {
        label: 'CreateStandard',
        keywords: createKeywords,
        value: `${APP_LINKS.Standard}/add`,
    },
    {
        label: 'ViewHistory',
        keywords: viewKeywords,
        value: `${APP_LINKS.History}`,
    },
    {
        label: 'ViewNotifications',
        keywords: viewKeywords,
        value: `${APP_LINKS.Notifications}`,
    },
    {
        label: 'ViewProfile',
        keywords: viewKeywords,
        value: `${APP_LINKS.Profile}`,
    },
    {
        label: 'ViewSettings',
        keywords: viewKeywords,
        value: `${APP_LINKS.Settings}`,
    },
    {
        label: 'SearchApi',
        keywords: searchKeywords,
        value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Apis}`,
    },
    {
        label: 'SearchNote',
        keywords: searchKeywords,
        value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Notes}`,
    },
    {
        label: 'SearchOrganization',
        keywords: searchKeywords,
        value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Organizations}`,
    },
    {
        label: 'SearchQuestion',
        keywords: searchKeywords,
        value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Questions}`,
    },
    {
        label: 'SearchProject',
        keywords: searchKeywords,
        value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Projects}`,
    },
    {
        label: 'SearchQuestion',
        keywords: searchKeywords,
        value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Questions}`,
    },
    {
        label: 'SearchRoutine',
        keywords: searchKeywords,
        value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Routines}`,
    },
    {
        label: 'SearchSmartContract',
        keywords: searchKeywords,
        value: `${APP_LINKS.Search}?type=${SearchPageTabOption.SmartContracts}`,
    },
    {
        label: 'SearchStandard',
        keywords: searchKeywords,
        value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Standards}`,
    },
    {
        label: 'SearchUser',
        keywords: searchKeywords,
        value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Users}`,
    },
    {
        label: 'SearchApiAdvanced',
        keywords: searchAdvancedKeywords,
        value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Apis}&advanced=true`,
    },
    {
        label: 'SearchOrganizationAdvanced',
        keywords: searchAdvancedKeywords,
        value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Organizations}&advanced=true`,
    },
    {
        label: 'SearchProjectAdvanced',
        keywords: searchAdvancedKeywords,
        value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Projects}&advanced=true`,
    },
    {
        label: 'SearchQuestionAdvanced',
        keywords: searchAdvancedKeywords,
        value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Questions}&advanced=true`,
    },
    {
        label: 'SearchRoutineAdvanced',
        keywords: searchAdvancedKeywords,
        value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Routines}&advanced=true`,
    },
    {
        label: 'SearchSmartContractAdvanced',
        keywords: searchAdvancedKeywords,
        value: `${APP_LINKS.Search}?type=${SearchPageTabOption.SmartContracts}&advanced=true`,
    },
    {
        label: 'SearchStandardAdvanced',
        keywords: searchAdvancedKeywords,
        value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Standards}&advanced=true`,
    },
    {
        label: 'SearchUserAdvanced',
        keywords: searchAdvancedKeywords,
        value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Users}&advanced=true`,
    },
    {
        label: 'SearchRun',
        keywords: searchKeywords,
        value: `${APP_LINKS.HistorySearch}?type=${HistorySearchPageTabOption.Runs}`,
    },
    {
        label: 'SearchView',
        keywords: [...searchKeywords, 'SearchViewed'],
        value: `${APP_LINKS.HistorySearch}?type=${HistorySearchPageTabOption.Viewed}`,
    },
    {
        label: 'SearchStar',
        keywords: [...searchKeywords, 'SearchBookmarked'],
        value: `${APP_LINKS.HistorySearch}?type=${HistorySearchPageTabOption.Bookmarked}`,
    },
    {
        label: 'SearchRunAdvanced',
        keywords: searchAdvancedKeywords,
        value: `${APP_LINKS.HistorySearch}?type=${HistorySearchPageTabOption.Runs}&advanced=true`,
    },
    {
        label: 'SearchViewAdvanced',
        keywords: [...searchAdvancedKeywords, 'SearchViewedAdvanced'],
        value: `${APP_LINKS.HistorySearch}?type=${HistorySearchPageTabOption.Viewed}&advanced=true`,
    },
    {
        label: 'SearchStarAdvanced',
        keywords: [...searchAdvancedKeywords, 'SearchBookmarkedAdvanced'],
        value: `${APP_LINKS.HistorySearch}?type=${HistorySearchPageTabOption.Bookmarked}&advanced=true`,
    },
    // { //TODO should be possible to replicate with normal advanced search
    //     label: 'Search your actively developing projects and routines',
    //     link: `${APP_LINKS.DevelopSearch}?type=${DevelopSearchPageTabOption.InProgress}`,
    // },
    // {
    //     label: 'Search your completed projects and routines',
    //     link: `${APP_LINKS.DevelopSearch}?type=${DevelopSearchPageTabOption.Completed}`,
    // },
    // {
    //     label: 'Search your actively developing projects and routines advanced',
    //     link: `${APP_LINKS.DevelopSearch}?type=${DevelopSearchPageTabOption.InProgress}&advanced=true`,
    // },
    // {
    //     label: 'Search your completed projects and routines advanced',
    //     link: `${APP_LINKS.DevelopSearch}?type=${DevelopSearchPageTabOption.Completed}&advanced=true`,
    // },
    {
        label: `BeginnersGuide`,
        keywords: viewKeywords,
        value: `${APP_LINKS.Welcome}`,
    },
    {
        label: 'Faq',
        keywords: viewKeywords,
        value: `${APP_LINKS.FAQ}`,
    },
    {
        label: 'Tutorial',
        keywords: viewKeywords,
        value: `${APP_LINKS.Tutorial}`,
    },
]

/**
 * Navigation shortcuts that can appear in the main search bar or command palette.
 */
export const shortcuts: ShortcutItem[] = [
    {
        label: 'Create new api',
        link: `${APP_LINKS.Api}/add`,
    },
    {
        label: 'Create new note',
        link: `${APP_LINKS.Note}/add`,
    },
    {
        label: 'Create new organization',
        link: `${APP_LINKS.Organization}/add`,
    },
    {
        label: 'Create new project',
        link: `${APP_LINKS.Project}/add`,
    },
    {
        label: 'Create new question',
        link: `${APP_LINKS.Question}/add`,
    },
    {
        label: 'Create new reminder',
        link: `${APP_LINKS.Reminder}/add`,
    },
    {
        label: 'Create new routine',
        link: `${APP_LINKS.Routine}/add`,
    },
    {
        label: 'Create new smart contract',
        link: `${APP_LINKS.SmartContract}/add`,
    },
    {
        label: 'Create new standard',
        link: `${APP_LINKS.Standard}/add`,
    },
    {
        label: 'View history page',
        link: `${APP_LINKS.History}`,
    },
    {
        label: 'View profile page',
        link: `${APP_LINKS.Profile}`,
    },
    {
        label: 'View settings page',
        link: `${APP_LINKS.Settings}`,
    },
    {
        label: 'Search organizations',
        link: `${APP_LINKS.Search}?type=${SearchPageTabOption.Organizations}`,
    },
    {
        label: 'Search projects',
        link: `${APP_LINKS.Search}?type=${SearchPageTabOption.Projects}`,
    },
    {
        label: 'Search routines',
        link: `${APP_LINKS.Search}?type=${SearchPageTabOption.Routines}`,
    },
    {
        label: 'Search standards',
        link: `${APP_LINKS.Search}?type=${SearchPageTabOption.Standards}`,
    },
    {
        label: 'Search users',
        link: `${APP_LINKS.Search}?type=${SearchPageTabOption.Users}`,
    },
    {
        label: 'Search organizations advanced',
        link: `${APP_LINKS.Search}?type=${SearchPageTabOption.Organizations}&advanced=true`,
    },
    {
        label: 'Search projects advanced',
        link: `${APP_LINKS.Search}?type=${SearchPageTabOption.Projects}&advanced=true`,
    },
    {
        label: 'Search routines advanced',
        link: `${APP_LINKS.Search}?type=${SearchPageTabOption.Routines}&advanced=true`,
    },
    {
        label: 'Search standards advanced',
        link: `${APP_LINKS.Search}?type=${SearchPageTabOption.Standards}&advanced=true`,
    },
    {
        label: 'Search users advanced',
        link: `${APP_LINKS.Search}?type=${SearchPageTabOption.Users}&advanced=true`,
    },
    {
        label: 'Search runs',
        link: `${APP_LINKS.HistorySearch}?type=${HistorySearchPageTabOption.Runs}`,
    },
    {
        label: 'Search viewed',
        link: `${APP_LINKS.HistorySearch}?type=${HistorySearchPageTabOption.Viewed}`,
    },
    {
        label: 'Search bookmarked',
        link: `${APP_LINKS.HistorySearch}?type=${HistorySearchPageTabOption.Bookmarked}`,
    },
    {
        label: 'Search runs advanced',
        link: `${APP_LINKS.HistorySearch}?type=${HistorySearchPageTabOption.Runs}&advanced=true`,
    },
    {
        label: 'Search viewed advanced',
        link: `${APP_LINKS.HistorySearch}?type=${HistorySearchPageTabOption.Viewed}&advanced=true`,
    },
    {
        label: 'Search bookmarked advanced',
        link: `${APP_LINKS.HistorySearch}?type=${HistorySearchPageTabOption.Bookmarked}&advanced=true`,
    },
    // { //TODO should be possible to replicate with normal advanced search
    //     label: 'Search your actively developing projects and routines',
    //     link: `${APP_LINKS.DevelopSearch}?type=${DevelopSearchPageTabOption.InProgress}`,
    // },
    // {
    //     label: 'Search your completed projects and routines',
    //     link: `${APP_LINKS.DevelopSearch}?type=${DevelopSearchPageTabOption.Completed}`,
    // },
    // {
    //     label: 'Search your actively developing projects and routines advanced',
    //     link: `${APP_LINKS.DevelopSearch}?type=${DevelopSearchPageTabOption.InProgress}&advanced=true`,
    // },
    // {
    //     label: 'Search your completed projects and routines advanced',
    //     link: `${APP_LINKS.DevelopSearch}?type=${DevelopSearchPageTabOption.Completed}&advanced=true`,
    // },
    {
        label: `Beginner's Guide`,
        link: `${APP_LINKS.Welcome}`,
    },
    {
        label: 'FAQ',
        link: `${APP_LINKS.FAQ}`,
    },
    {
        label: 'Tutorial',
        link: `${APP_LINKS.Tutorial}`,
    },
]

/**
 * Shape shortcuts to match AutoCompleteListItem format.
 */
export const shortcutsItems: ShortcutOption[] = shortcuts.map(({ label, link }) => ({
    __typename: "Shortcut",
    label,
    id: link,
}))

/**
 * Action shortcuts that can appear in the main search bar or command palette. 
 * Instead of taking you to a page, they perform an action (e.g. clear search history).
 */
export const actions: ActionItem[] = [
    {
        label: 'Clear search history',
        id: 'clear-search-history',
        canPerform: () => true,
    },
    {
        label: 'Activate dark mode',
        id: 'activate-dark-mode',
        canPerform: (session: Session) => getCurrentUser(session).theme !== 'dark',
    },
    {
        label: 'Activate light mode',
        id: 'activate-light-mode',
        canPerform: (session: Session) => getCurrentUser(session).theme !== 'light',
    },
]

/**
 * Shape actions to match AutoCompleteListItem format.
 */
export const actionsItems: ActionOption[] = actions.map(({ canPerform, id, label }) => ({
    __typename: "Action",
    canPerform,
    id,
    label,
}))

/**
 * Maps action ids to their corresponding action. 
 * Actions cannot be stored in the options themselves because localStorage cannot store functions.
 */
export const performAction = async (option: ActionOption, session: Session): Promise<void> => {
    switch (option.id) {
        case 'clear-search-history':
            clearSearchHistory(session);
            break;
        case 'activate-dark-mode':
            documentNodeWrapper<User, ProfileUpdateInput>({
                node: userProfileUpdate,
                input: { theme: 'dark' },
                onSuccess: () => { PubSub.get().publishTheme('dark'); },
                onError: (error) => { PubSub.get().publishSnack({ messageKey: errorToCode(error), severity: SnackSeverity.Error, data: error }); }
            })
            break;
        case 'activate-light-mode':
            documentNodeWrapper<User, ProfileUpdateInput>({
                node: userProfileUpdate,
                input: { theme: 'light' },
                onSuccess: () => { PubSub.get().publishTheme('light'); },
                onError: (error) => { PubSub.get().publishSnack({ messageKey: errorToCode(error), severity: SnackSeverity.Error, data: error }); }
            })
            break;
    }
}