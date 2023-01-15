import { useCallback, useMemo } from "react";
import { ObjectPageProps } from "../types";
import { ObjectDialogAction } from "components/dialogs/types";
import { useLocation } from '@shared/route';
import { APP_LINKS, Organization, ProjectVersion, RoutineVersion, Session, StandardVersion, User } from "@shared/consts";
import { lazily } from "react-lazily";
import { ObjectType, parseSearchParams, PubSub, uuidToBase36 } from "utils";
import { PageContainer, ReportsView, SnackSeverity } from "components";

const { OrganizationCreate, OrganizationUpdate, OrganizationView } = lazily(() => import('../../components/views/Organization'));
const { ProjectCreate, ProjectUpdate, ProjectView } = lazily(() => import('../../components/views/Project'));
const { RoutineCreate, RoutineUpdate, RoutineView } = lazily(() => import('../../components/views/Routine'));
const { StandardCreate, StandardUpdate, StandardView } = lazily(() => import('../../components/views/Standard'));

export interface CreatePageProps {
    onCancel: () => void;
    onCreated: (item: { type: string, id: string }) => void;
    session: Session;
    zIndex: number;
}

export interface UpdatePageProps {
    onCancel: () => void;
    onUpdated: (item: { type: string, id: string }) => void;
    session: Session;
    zIndex: number;
}

export interface ViewPageProps {
    /**
     * Any data about the object which is already known, 
     * such as its name. Can be displayed while fetching the full object
     */
    partialData?: Partial<Organization & ProjectVersion & RoutineVersion & StandardVersion & User>
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
const typeMap: { [key in APP_LINKS]?: ObjectType } = {
    [APP_LINKS.Organization]: 'Organization',
    [APP_LINKS.Project]: 'Project',
    [APP_LINKS.Routine]: 'Routine',
    [APP_LINKS.Standard]: 'Standard',
}

/**
 * Maps object types to dialog titles
 */
const titleMap: { [key in ObjectType]?: string } = {
    'Organization': 'Organization',
    'Project': 'Project',
    'Routine': 'Routine',
    'Standard': 'Standard',
}

/**
 * Maps object types to create components
 */
const createMap: { [key in ObjectType]?: (props: CreatePageProps) => JSX.Element } = {
    'Organization': OrganizationCreate,
    'Project': ProjectCreate,
    'Routine': RoutineCreate,
    'Standard': StandardCreate,
}

/**
 * Maps object types to update components
 */
const updateMap: { [key in ObjectType]?: (props: UpdatePageProps) => JSX.Element } = {
    'Organization': OrganizationUpdate,
    'Project': ProjectUpdate,
    'Routine': RoutineUpdate,
    'Standard': StandardUpdate,
}

/**
 * Maps object types to view components
 */
const viewMap: { [key in ObjectType]?: (props: ViewPageProps) => JSX.Element } = {
    'Organization': OrganizationView,
    'Project': ProjectView,
    'Routine': RoutineView,
    'Standard': StandardView,
}

export const ObjectPage = ({
    session,
}: ObjectPageProps) => {
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
                    severity: SnackSeverity.Success,
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

    const displayedPage = useMemo<JSX.Element | undefined>(() => {
        if (!objectType) return undefined;
        // If page type is View, display the view page
        // Also display the view page for multi-step routines, since this has special logic
        const searchParams = parseSearchParams();
        if (pageType === PageType.View || searchParams.build === true) {
            const View = viewMap[objectType];
            document.title = `View ${titleMap[objectType]}`;
            return View && <View session={session} zIndex={200} />
        }
        if (pageType === PageType.Create) {
            const Create = createMap[objectType];
            document.title = `Create ${titleMap[objectType]}`;
            return (Create && <Create
                onCancel={() => onAction(ObjectDialogAction.Cancel)}
                onCreated={(data) => onAction(ObjectDialogAction.Add, data)}
                session={session}
                zIndex={200}
            />)
        }
        if (pageType === PageType.Update) {
            const Update = updateMap[objectType];
            document.title = `Update ${titleMap[objectType]}`;
            return (Update && <Update
                onCancel={() => onAction(ObjectDialogAction.Cancel)}
                onUpdated={(data) => onAction(ObjectDialogAction.Save, data)}
                session={session}
                zIndex={200}
            />)
        }
        if (pageType === PageType.Reports) {
            document.title = `Reports | ${titleMap[objectType]}`;
            return <ReportsView session={session} />
        }
        const View = viewMap[objectType];
        document.title = `View ${titleMap[objectType]}`;
        return View && <View session={session} zIndex={200} />
    }, [objectType, onAction, pageType, session]);

    return (
        <PageContainer sx={{ paddingLeft: 0, paddingRight: 0 }}>
            {displayedPage}
        </PageContainer>
    )
}