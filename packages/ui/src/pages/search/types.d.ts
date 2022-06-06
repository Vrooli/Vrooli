import { DocumentNode } from "graphql";
import { MouseEvent } from "react";
import { Session } from "types";
import { ObjectType } from "utils";

export interface SearchQueryVariables {
    input: UserSearchInput;
}

export interface BaseSearchPageProps {
    itemKeyPrefix: string;
    title?: string | null;
    searchPlaceholder?: string;
    query: DocumentNode;
    take?: number; // Number of items to fetch per page
    objectType: ObjectType;
    onObjectSelect: (objectData: any) => void; // Passes all object data to the parent, so the known information can be displayed while more details are queried
    showAddButton?: boolean; // Displays add button next to title
    onAddClick?: (ev: MouseEvent<any>) => void; // Callback when add button is clicked
    popupButtonText?: string; // Button that appears when scrolled past a certain point, prompting the user to create a new object
    popupButtonTooltip?: string; // Tooltip for the popup button
    onPopupButtonClick?: (ev: MouseEvent<any>) => void; // Called when the popup button is clicked
    session: Session;
    where?: any; // Additional query variables
}

export interface SearchPageBaseProps {
    session: Session;
}

export interface SearchOrganizationsPageProps extends SearchPageBaseProps {}

export interface SearchProjectsPageProps extends SearchPageBaseProps {}

export interface SearchRoutinesPageProps extends SearchPageBaseProps {}

export interface SearchStandardsPageProps extends SearchPageBaseProps {}

export interface SearchUsersPageProps extends SearchPageBaseProps {}