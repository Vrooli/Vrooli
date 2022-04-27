import { ListOrganization, ListProject, ListRoutine, ListStandard, ListUser, Session } from "types";
import { OrganizationListItem, ProjectListItem, RoutineListItem, StandardListItem, UserListItem } from 'components';
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

/**
 * Converts a list of objects to a list of ListItems
 * @param objects The list of objects.
 * @param session Current session
 * @param keyPrefix Each list item's key will be `${keyPrefix}-${id}`.
 * @param onClick Function to call when a list item is clicked.
 * @returns A list of ListItems
 */
export function listToListItems(
    objects: readonly (ListOrganization | ListProject | ListRoutine | ListStandard | ListUser)[],
    session: Session,
    keyPrefix: string,
    onClick: (item: ListOrganization | ListProject | ListRoutine | ListStandard | ListUser, event: React.MouseEvent<HTMLElement>) => void,
): JSX.Element[] {
    let listItems: JSX.Element[] = [];
    for (let i = 0; i < objects.length; i++) {
        const curr = objects[i];
        switch (curr.__typename) {
            case 'Organization':
                listItems.push(<OrganizationListItem
                    key={`${keyPrefix}-${curr.id}`}
                    index={i}
                    session={session}
                    data={curr as ListOrganization}
                    onClick={(e) => onClick(curr, e)}
                />);
                break;
            case 'Project':
                listItems.push(<ProjectListItem
                    key={`${keyPrefix}-${curr.id}`}
                    index={i}
                    session={session}
                    data={curr as ListProject}
                    onClick={(e) => onClick(curr, e)}
                />);
                break;
            case 'Routine':
                listItems.push(<RoutineListItem
                    key={`${keyPrefix}-${curr.id}`}
                    index={i}
                    session={session}
                    data={curr as ListRoutine}
                    onClick={(e) => onClick(curr, e)}
                />);
                break;
            case 'Standard':
                listItems.push(<StandardListItem
                    key={`${keyPrefix}-${curr.id}`}
                    index={i}
                    session={session}
                    data={curr as ListStandard}
                    onClick={(e) => onClick(curr, e)}
                />);
                break;
            case 'User':
                listItems.push(<UserListItem
                    key={`${keyPrefix}-${curr.id}`}
                    index={i}
                    session={session}
                    data={curr as ListUser}
                    onClick={(e) => onClick(curr, e)}
                />);
                break;
            default:
                break;
        }
    }
    return listItems;
}