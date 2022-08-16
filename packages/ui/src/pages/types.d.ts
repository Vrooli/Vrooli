import { Organization, Project, Routine, Session, Standard, User } from 'types';

export interface CreatePageProps {
    onCancel: () => void;
    onCreated: (item: { id: string }) => void;
    session: Session;
    zIndex: number;
}

export interface UpdatePageProps {
    onCancel: () => void;
    onUpdated: (item: { id: string }) => void;
    session: Session;
    zIndex: number;
}

export interface ViewPageProps {
    /**
     * Any data about the object which is already known, 
     * such as its name. Can be displayed while fetching the full object
     */
    partialData?: Partial<Organization & Project & Routine & Standard & User>
    session: Session;
    zIndex: number;
}

export interface  ObjectPageProps {
    session: Session;
    Create: (props: CreatePageProps) => JSX.Element;
    Update: (props: UpdatePageProps) => JSX.Element;
    View: (props: ViewPageProps) => JSX.Element;
}

export interface SearchPageProps {
    session: Session;
}

export interface SettingsPageProps {
    session: Session;
}

export interface StartPageProps {
    session: Session;
}