import { APP_LINKS } from "@shared/consts";
import { AutocompleteOption } from "types";
import { DevelopSearchPageTabOption, HistorySearchPageTabOption, SearchPageTabOption } from "utils/display";

export interface ShortcutItem {
    label: string;
    link: string;
}
/**
 * Shortcuts that can appear in the main search bar or command palette.
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
        label: 'Create new single-step routine',
        link: `${APP_LINKS.Routine}/add`,
    },
    {
        label: 'Create new multi-step routine',
        link: `${APP_LINKS.Routine}/add?build=true`,
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
        link: `${APP_LINKS.Search}?type=${SearchPageTabOption.Routines}`,
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
]
// Shape shortcuts to match AutoCompleteListItem format
export const shortcutsItems: AutocompleteOption[] = shortcuts.map(({ label, link }) => ({
    __typename: "Shortcut",
    label,
    id: link,
}))