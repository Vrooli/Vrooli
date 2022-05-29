import { AutocompleteOption, ListOrganization, ListProject, ListRoutine, ListRun, ListStandard, ListStar, ListUser, ListView, Session } from "types";
import { OrganizationListItem, ProjectListItem, RoutineListItem, RunListItem, StandardListItem, UserListItem } from 'components';
import { getTranslation, getUserLanguages } from "./translationTools";
import { ObjectListItemProps } from "components/lists/types";
import { Theme } from "@mui/material";

/**
 * Gets label for List, either from its 
 * name or handle
 * @param o Organization object
 * @param languages User languages
 * @returns label
 */
export const organizationOptionLabel = (o: ListOrganization, languages: readonly string[]) => getTranslation(o, 'name', languages, true) ?? o.handle ?? '';

/**
 * Gets label for project, either from its 
 * name or handle
 * @param o Project object
 * @param languages User languages
 * @returns label
 */
export const projectOptionLabel = (o: ListProject, languages: readonly string[]) => getTranslation(o, 'name', languages, true) ?? o.handle ?? '';

/**
* Gets label for routine, from its title
* @param o Routine object
* @param languages User languages
* @returns label
*/
export const routineOptionLabel = (o: ListRoutine, languages: readonly string[]) => getTranslation(o, 'title', languages, true) ?? '';

/**
* Gets label for routine, from its name
* @param o Standard object
* @param languages User languages
* @returns label
*/
export const standardOptionLabel = (o: ListStandard, languages: readonly string[]) => o.name ?? '';

/**
* Gets label for user, either from its 
* name or handle
* @param o User object
* @param languages User languages
* @returns label
*/
export const userOptionLabel = (o: ListUser, languages: readonly string[]) => o.name ?? o.handle ?? '';

/**
 * Gets label for a single object
 * @param object Either an Organization, Project, Routine, Standard or User
 * @param languages User languages
 * @returns label
 */
export const getListItemLabel = (
    object: ListOrganization | ListProject | ListRoutine | ListStandard | ListUser,
    languages?: readonly string[]
) => {
    const lang = languages ?? getUserLanguages();
    switch (object.__typename) {
        case 'Organization':
            return organizationOptionLabel(object, lang);
        case 'Project':
            return projectOptionLabel(object, lang);
        case 'Routine':
            return routineOptionLabel(object, lang);
        case 'Standard':
            return standardOptionLabel(object, lang);
        case 'User':
            return userOptionLabel(object, lang);
        default:
            return '';
    }
};

/**
 * Converts a list of GraphQL objects to a list of autocomplete information.
 * @param objects The list of Organizations, Projects, Routines, Standards, and/or Users.
 * @param languages User languages
 * @returns The list of autocomplete information. Each object has the following shape: 
 * {
 *  id: The ID of the object.
 *  label: The label of the object.
 *  stars: The number of stars the object has.
 * }
 */
export function listToAutocomplete(
    objects: readonly (ListOrganization | ListProject | ListRoutine | ListStandard | ListUser)[],
    languages: readonly string[]
): AutocompleteOption[] {
    return objects.map(o => ({
        __typename: o.__typename,
        id: o.id,
        label: getListItemLabel(o, languages),
        stars: o.stars ?? 0
    }));
}

/**
 * Maps __typename to the corresponding ListItem component.
 */
export const listItemMap: { [x: string] : (props: ObjectListItemProps<any>) => JSX.Element } = {
    'Organization': (props) => <OrganizationListItem {...props} />,
    'Project': (props) => <ProjectListItem {...props} />,
    'Routine': (props) => <RoutineListItem {...props} />,
    'Run': (props) => <RunListItem {...props} />,
    'Standard': (props) => <StandardListItem {...props} />,
    'User': (props) => <UserListItem {...props} />,
};

type ListItem = ListOrganization | ListProject | ListRoutine | ListRun | ListStandard | ListStar | ListUser | ListView;

export interface ListToListItemProps {
    /**
     * List of dummy items types to display while loading
     */
    dummyItems?: string[];
    /**
     * The list of item data
     */
    items?: readonly ListItem[],
    /**
     * Each list item's key will be `${keyPrefix}-${id}`
     */
    keyPrefix: string,
    /**
     * Whether the list is loading
     */
    loading: boolean,
    /**
     * Function to call when a list item is clicked
     */
    onClick?: (item: ListItem, event: React.MouseEvent<HTMLElement>) => void,
    /**
     * Current session
     */
    session: Session,
    /**
     * Tooltip text to display when hovering over a list item
     */
    tooltip?: string,
}

/**
 * Converts a list of objects to a list of ListItems
 * @returns A list of ListItems
 */
export function listToListItems({
    dummyItems,
    keyPrefix,
    items,
    loading,
    onClick,
    session,
    tooltip,
}: ListToListItemProps): JSX.Element[] {
    let listItems: JSX.Element[] = [];
    // If loading, display dummy items
    if (loading) {
        if (!dummyItems) return listItems;
        for (let i = 0; i < dummyItems.length; i++) {
            if (dummyItems[i] in listItemMap) {
                const CurrItem = listItemMap[dummyItems[i]];
                listItems.push(<CurrItem 
                    key={`${keyPrefix}-${i}`} 
                    data={null}
                    index={i}
                    loading={true}
                    session={session}
                    tooltip={tooltip}
                />);
            }
        }
    }
    if (!items) return listItems;
    for (let i = 0; i < items.length; i++) {
        let curr = items[i];
        // If "View" or "Star" item, display the object it points to
        if (curr.__typename === 'View' || curr.__typename === 'Star') {
            curr = (curr as ListStar | ListView).to;
        }
        // Create common props
        const commonProps = {
            key: `${keyPrefix}-${curr.id}`,
            data: curr,
            index: i,
            loading: loading,
            session,
            onClick: onClick ? (e) => onClick(curr, e) : undefined,
            tooltip: tooltip,
        }
        if (curr.__typename in listItemMap) {
            const CurrItem = listItemMap[curr.__typename];
            listItems.push(<CurrItem {...commonProps} />);
        }
    }
    return listItems;
}

/**
 * Determines background color for a list item. Alternates between 
 * two colors.
 * @param index Index of list item
 * @param palette MUI theme palette
 * @returns String of background color
 */
export const listItemColor = (index: number, palette: Theme['palette']): string => {
    const lightColors = [palette.background.default, palette.background.paper];
    const darkColors = [palette.background.default, palette.background.paper];
    return (palette.mode === 'light') ? lightColors[index % lightColors.length] : darkColors[index % darkColors.length];
}

/**
 * Color options for placeholder icon
 * [background color, silhouette color]
 */
const placeholderColors: [string, string][] = [
    ["#197e2c", "#b5ffc4"],
    ["#b578b6", "#fecfea"],
    ["#4044d6", "#e1c7f3"],
    ["#d64053", "#fbb8c5"],
    ["#d69440", "#e5d295"],
    ["#40a4d6", "#79e0ef"],
    ["#6248e4", "#aac3c9"],
    ["#8ec22c", "#cfe7b4"],
]

/**
 * Finds a random color for a placeholder icon
 * @returns [background color code, silhouette color code]
 */
export const placeholderColor = (): [string, string] => {
    return placeholderColors[Math.floor(Math.random() * placeholderColors.length)];
}