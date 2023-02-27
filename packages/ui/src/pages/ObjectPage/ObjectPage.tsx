import { useCallback, useEffect, useMemo, useState } from "react";
import { ObjectPageProps } from "../types";
import { ObjectDialogAction } from "components/dialogs/types";
import { parseSearchParams, useLocation } from '@shared/route';
import { Api, APP_LINKS, GqlModelType, Note, Organization, ProjectVersion, RoutineVersion, Session, SmartContractVersion, StandardVersion, User } from "@shared/consts";
import { lazily } from "react-lazily";
import { ObjectType, PubSub, uuidToBase36 } from "utils";
import { PageContainer, ReportsView } from "components";
import { useTranslation } from "react-i18next";

export interface CreatePageProps {
    onCancel: () => void;
    onCreated: (item: { __typename: `${GqlModelType}`, id: string }) => void;
    session: Session;
    zIndex: number;
}

export interface UpdatePageProps {
    onCancel: () => void;
    onUpdated: (item: { __typename: `${GqlModelType}`, id: string }) => void;
    session: Session;
    zIndex: number;
}

export interface ViewPageProps {
    /**
     * Any data about the object which is already known, 
     * such as its name. Can be displayed while fetching the full object
     */
    partialData?: Partial<Api & Note & Organization & ProjectVersion & RoutineVersion & SmartContractVersion & StandardVersion & User>
    session: Session;
    zIndex: number;
}

export interface ReportsPageProps { }

enum PageType {
    Create = 'Create',
    Update = 'Update',
    View = 'View',
    Reports = 'Reports',
}

/**
 * Maps links to object types
 */
const typeMap = {
    [APP_LINKS.Api]: 'Api',
    [APP_LINKS.Note]: 'Note',
    [APP_LINKS.Organization]: 'Organization',
    [APP_LINKS.Project]: 'Project',
    [APP_LINKS.Question]: 'Question',
    [APP_LINKS.Reminder]: 'Reminder',
    [APP_LINKS.Routine]: 'Routine',
    [APP_LINKS.SmartContract]: 'SmartContract',
    [APP_LINKS.Standard]: 'Standard',
} as const

/**
 * Maps object types to create components
 */
const createMap = {
    Api: async () => (await import('../../components/views/Api')).ApiCreate,
    Note: async () => (await import('../../components/views/Note')).NoteCreate,
    Organization: async () => (await import('../../components/views/Organization')).OrganizationCreate,
    Project: async () => (await import('../../components/views/Project')).ProjectCreate,
    Question: async () => (await import('../../components/views/Question')).QuestionCreate,
    Reminder: async () => (await import('../../components/views/Reminder')).ReminderCreate,
    Routine: async () => (await import('../../components/views/Routine')).RoutineCreate,
    SmartContract: async () => (await import('../../components/views/SmartContract')).SmartContractCreate,
    Standard: async () => (await import('../../components/views/Standard')).StandardCreate,
}

/**
 * Maps object types to update components
 */
const updateMap = {
    Api: async () => (await import('../../components/views/Api')).ApiUpdate,
    Note: async () => (await import('../../components/views/Note')).NoteUpdate,
    Organization: async () => (await import('../../components/views/Organization')).OrganizationUpdate,
    Project: async () => (await import('../../components/views/Project')).ProjectUpdate,
    Question: async () => (await import('../../components/views/Question')).QuestionUpdate,
    Reminder: async () => (await import('../../components/views/Reminder')).ReminderUpdate,
    Routine: async () => (await import('../../components/views/Routine')).RoutineUpdate,
    SmartContract: async () => (await import('../../components/views/SmartContract')).SmartContractUpdate,
    Standard: async () => (await import('../../components/views/Standard')).StandardUpdate,
}

/**
 * Maps object types to view components
 */
const viewMap = {
    Api: async () => (await import('../../components/views/Api')).ApiView,
    Note: async () => (await import('../../components/views/Note')).NoteView,
    Organization: async () => (await import('../../components/views/Organization')).OrganizationView,
    Project: async () => (await import('../../components/views/Project')).ProjectView,
    Question: async () => (await import('../../components/views/Question')).QuestionView,
    Reminder: async () => (await import('../../components/views/Reminder')).ReminderView,
    Routine: async () => (await import('../../components/views/Routine')).RoutineView,
    SmartContract: async () => (await import('../../components/views/SmartContract')).SmartContractView,
    Standard: async () => (await import('../../components/views/Standard')).StandardView,
}

export const ObjectPage = ({
    session,
}: ObjectPageProps) => {
    const { t } = useTranslation();
    const [location, setLocation] = useLocation();

    // Determine if page should be displayed as a dialog or full page. 
    // Also checks if the create, update, or view page should be shown
    const { hasPreviousPage, objectType, pageType } = useMemo(() => {
        const objectType = typeMap['/' + location.split('/')[1]];
        // Read session storage to check previous page.
        const hasPreviousPage = Boolean(sessionStorage.getItem('lastPath'));
        // Determine if create, update, or view page should be shown using the URL
        let pageType: PageType = PageType.View;
        if (location.endsWith('/add')) pageType = PageType.Create;
        else if (location.includes('/reports/')) pageType = PageType.Reports;
        else if (location.includes('/edit/')) pageType = PageType.Update;
        return { hasPreviousPage, objectType, pageType };
    }, [location]);

    const onAction = useCallback((action: ObjectDialogAction, item?: { __typename: string, id: string }) => {
        // Only navigate back if there is a previous page
        const pageRoot = window.location.pathname.split('/')[1];
        switch (action) {
            case ObjectDialogAction.Add:
                setLocation(`${uuidToBase36(item?.id ?? '')}`, { replace: !hasPreviousPage });
                PubSub.get().publishSnack({
                    message: `${item?.__typename ?? ''} created!`,
                    severity: 'Success',
                    buttonText: 'Create another',
                    buttonClicked: () => { setLocation(`add`); },
                } as any) //TODO
                break;
            case ObjectDialogAction.Cancel:
            case ObjectDialogAction.Close:
                if (hasPreviousPage) window.history.back();
                else setLocation(APP_LINKS.Home);
                break;
            case ObjectDialogAction.Edit:
                setLocation(`${pageRoot}/edit/${uuidToBase36(item?.id ?? '')}`);
                break;
            case ObjectDialogAction.Save:
                if (hasPreviousPage) window.history.back();
                else setLocation(APP_LINKS.Home);
                break;
        }
    }, [hasPreviousPage, setLocation]);

    const [displayedPage, setDisplayedPage] = useState<JSX.Element | undefined>(undefined);
    useEffect(() => {
        const fetchDisplayedPage = async () => {
            if (!objectType) return undefined;
            const searchParams = parseSearchParams();
            // If page type is reports, display reports page
            if (pageType === PageType.Reports) {
                document.title = t(`Reports`) + '|' + t(objectType, { count: 1 });
                setDisplayedPage(<ReportsView session={session} />);
                return;
            }
            // Otherwise, determine which object map to use (create, update, or view)
            let pageMap: Record<string, any>;
            // View page for View pages OR multi-step routines
            if (pageType === PageType.View || searchParams.build === true) pageMap = viewMap;
            // Create page for Create pages
            else if (pageType === PageType.Create) pageMap = createMap;
            // Update page for Update pages
            else if (pageType === PageType.Update) pageMap = updateMap;
            // Default to view page
            else pageMap = viewMap;
            // Use map to get the correct component
            const Page = await (pageMap as any)[objectType.replace('Version', '')];
            // If page is not found, display error
            if (!Page) {
                PubSub.get().publishSnack({ messageKey: 'PageNotFound', severity: 'Error' });
            }
            // Set displayed page
            setDisplayedPage(<Page
                onCancel={() => onAction(ObjectDialogAction.Cancel)}
                onUpdated={(data: any) => onAction(ObjectDialogAction.Save, data)}
                session={session}
                zIndex={200}
            />);
        };
        fetchDisplayedPage();
    }, [objectType, onAction, pageType, session, t]);

    return (
        <PageContainer sx={{ paddingLeft: 0, paddingRight: 0 }}>
            {displayedPage}
        </PageContainer>
    )
}