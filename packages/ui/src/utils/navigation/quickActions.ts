import { APP_LINKS, ProfileUpdateInput, Session, User } from "@shared/consts";
import { SnackSeverity } from "components";
import { userEndpoint } from "api/endpoints";
import { documentNodeWrapper, errorToCode } from "api/utils";
import { ActionOption, ShortcutOption } from "types";
import { getCurrentUser } from "utils/authentication";
import { clearSearchHistory, DevelopSearchPageTabOption, HistorySearchPageTabOption, SearchPageTabOption } from "utils/display";
import { PubSub } from "utils/pubsub";

export interface ShortcutItem {
    label: string;
    link: string;
}

export interface ActionItem {
    canPerform: (session: Session) => boolean;
    id: string;
    label: string;
}

/**
 * Navigation shortcuts that can appear in the main search bar or command palette.
 */
export const shortcuts: ShortcutItem[] = [
    {
        label: 'Create new organization',
        link: `${APP_LINKS.Organization}/add`,
    },
    {
        label: 'Create new project',
        link: `${APP_LINKS.Project}/add`,
    },
    {
        label: 'Create new routine',
        link: `${APP_LINKS.Routine}/add`,
    },
    {
        label: 'Create new standard',
        link: `${APP_LINKS.Standard}/add`,
    },
    {
        label: 'View learn dashboard',
        link: `${APP_LINKS.Learn}`,
    },
    {
        label: 'View research dashboard',
        link: `${APP_LINKS.Research}`,
    },
    {
        label: 'View develop dashboard',
        link: `${APP_LINKS.Develop}`,
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
        label: 'Search starred',
        link: `${APP_LINKS.HistorySearch}?type=${HistorySearchPageTabOption.Starred}`,
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
        label: 'Search starred advanced',
        link: `${APP_LINKS.HistorySearch}?type=${HistorySearchPageTabOption.Starred}&advanced=true`,
    },
    {
        label: 'Search your actively developing projects and routines',
        link: `${APP_LINKS.DevelopSearch}?type=${DevelopSearchPageTabOption.InProgress}`,
    },
    {
        label: 'Search your completed projects and routines',
        link: `${APP_LINKS.DevelopSearch}?type=${DevelopSearchPageTabOption.Completed}`,
    },
    {
        label: 'Search your actively developing projects and routines advanced',
        link: `${APP_LINKS.DevelopSearch}?type=${DevelopSearchPageTabOption.InProgress}&advanced=true`,
    },
    {
        label: 'Search your completed projects and routines advanced',
        link: `${APP_LINKS.DevelopSearch}?type=${DevelopSearchPageTabOption.Completed}&advanced=true`,
    },
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
                node: userEndpoint.profileUpdate[0],
                input: { theme: 'dark' },
                onSuccess: () => { PubSub.get().publishTheme('dark'); },
                onError: (error) => { PubSub.get().publishSnack({ messageKey: errorToCode(error), severity: SnackSeverity.Error, data: error }); }
            })
            break;
        case 'activate-light-mode':
            documentNodeWrapper<User, ProfileUpdateInput>({
                node: userEndpoint.profileUpdate[0],
                input: { theme: 'light' },
                onSuccess: () => { PubSub.get().publishTheme('light'); },
                onError: (error) => { PubSub.get().publishSnack({ messageKey: errorToCode(error), severity: SnackSeverity.Error, data: error }); }
            })
            break;
    }
}