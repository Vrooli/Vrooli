import { DocumentNode } from "graphql";
import { Session } from "types";

export interface SearchQueryVariables {
    input: UserSearchInput;
}

export interface SearchQueryVariablesInput<SortBy> {
    ids?: string[] | null;
    sortBy?: SortBy | null;
    searchString?: string | null;
    after?: string | null;
    take?: number | null;
}

export interface BaseSearchPageProps<DataType, SortBy> {
    title?: string | null;
    searchPlaceholder?: string;
    sortOptions: SearchSortBy<SortBy>[];
    defaultSortOption: SearchSortBy<SortBy>;
    query: DocumentNode;
    take?: number; // Number of items to fetch per page
    listItemFactory: (node: DataType, index: number) => JSX.Element;
    getOptionLabel: (option: any) => string;
    onObjectSelect: (objectData: any) => void; // Passes all object data to the parent, so the known information can be displayed while more details are queried
    popupButtonText?: string; // Button that appears when scrolled past a certain point, prompting the user to create a new object
    popupButtonTooltip?: string; // Tooltip for the popup button
    onPopupButtonClick?: () => void; // Called when the popup button is clicked
}

export interface SearchPageBaseProps {
    session: Session;
}

export interface SearchActorsPageProps extends SearchPageBaseProps {}

export interface SearchOrganizationsPageProps extends SearchPageBaseProps {}

export interface SearchProjectsPageProps extends SearchPageBaseProps {}

export interface SearchRoutinesPageProps extends SearchPageBaseProps {}

export interface SearchStandardsPageProps extends SearchPageBaseProps {}