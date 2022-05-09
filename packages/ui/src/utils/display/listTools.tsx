import { ListOrganization, ListProject, ListRoutine, ListRun, ListStandard, ListStar, ListUser, ListView, Session } from "types";
import { OrganizationListItem, ProjectListItem, RoutineListItem, RunListItem, StandardListItem, UserListItem } from 'components';
import { getTranslation, getUserLanguages } from "./translationTools";

export interface AutocompleteListItem {
    __typename: string;
    id: string;
    label: string | null;
    stars?: number;
}

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
): AutocompleteListItem[] {
    return objects.map(o => ({
        __typename: o.__typename,
        id: o.id,
        label: getListItemLabel(o, languages),
        stars: o.stars ?? 0
    }));
}

type ListItem = ListOrganization | ListProject | ListRoutine | ListRun | ListStandard | ListStar | ListUser | ListView;

/**
 * Converts a list of objects to a list of ListItems
 * @param objects The list of objects.
 * @param session Current session
 * @param keyPrefix Each list item's key will be `${keyPrefix}-${id}`.
 * @param onClick Function to call when a list item is clicked.
 * @returns A list of ListItems
 */
export function listToListItems(
    objects: readonly ListItem[],
    session: Session,
    keyPrefix: string,
    onClick: (item: ListItem, event: React.MouseEvent<HTMLElement>) => void,
): JSX.Element[] {
    console.log('listtolistitems start', objects)
    let listItems: JSX.Element[] = [];
    for (let i = 0; i < objects.length; i++) {
        let curr = objects[i];
        // If "View" or "Star" item, display the object it points to
        if (curr.__typename === 'View' || curr.__typename === 'Star') {
            console.log('was view or star a', curr)
            curr = (curr as ListStar | ListView).to;
            console.log('now is', curr)
        }
        // Create common props
        const commonProps = {
            key: `${keyPrefix}-${curr.id}`,
            index: i,
            session,
            onClick: (e) => onClick(curr, e)
        }
        switch (curr.__typename) {
            case 'Organization':
                listItems.push(<OrganizationListItem
                    {...commonProps}
                    data={curr as ListOrganization}
                />);
                break;
            case 'Project':
                listItems.push(<ProjectListItem
                    {...commonProps}
                    data={curr as ListProject}
                />);
                break;
            case 'Routine':
                listItems.push(<RoutineListItem
                    {...commonProps}
                    data={curr as ListRoutine}
                />);
                break;
            case 'Run':
                listItems.push(<RunListItem
                    {...commonProps}
                    data={curr as ListRun}
                />);
                break;
            case 'Standard':
                listItems.push(<StandardListItem
                    {...commonProps}
                    data={curr as ListStandard}
                />);
                break;
            case 'User':
                listItems.push(<UserListItem
                    {...commonProps}
                    data={curr as ListUser}
                />);
                break;
            default:
                break;
        }
    }
    return listItems;
}