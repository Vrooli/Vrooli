import { useCallback, useMemo } from "react";
import { Box } from "@mui/material"
import { BaseObjectDialog } from "components";
import { ObjectPageProps } from "../types";
import { ObjectDialogAction } from "components/dialogs/types";
import { useLocation } from '@shared/route';
import { APP_LINKS } from "@shared/consts";
import { lazily } from "react-lazily";
import { ObjectType } from "utils";
import { Organization, Project, Routine, Session, Standard, User } from "types";

const { OrganizationCreate, OrganizationUpdate, OrganizationView } = lazily(() => import('../../components/views/Organization'));
const { ProjectCreate, ProjectUpdate, ProjectView } = lazily(() => import('../../components/views/Project'));
const { RoutineCreate, RoutineUpdate, RoutineView } = lazily(() => import('../../components/views/Routine'));
const { StandardCreate, StandardUpdate, StandardView } = lazily(() => import('../../components/views/Standard'));

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

enum PageType {
    Create = 'Create',
    Update = 'Update',
    View = 'View',
}

/**
 * Maps links to object types
 */
const typeMap: { [key in APP_LINKS]?: ObjectType } = {
    [APP_LINKS.Organization]: ObjectType.Organization,
    [APP_LINKS.Project]: ObjectType.Project,
    [APP_LINKS.Routine]: ObjectType.Routine,
    [APP_LINKS.Standard]: ObjectType.Standard,
}

/**
 * Maps object types to dialog titles
 */
 const titleMap: { [key in ObjectType]?: string } = {
    [ObjectType.Organization]: 'Organization',
    [ObjectType.Project]: 'Project',
    [ObjectType.Routine]: 'Routine',
    [ObjectType.Standard]: 'Standard',
}

/**
 * Maps object types to create components
 */
const createMap: { [key in ObjectType]?: (props: CreatePageProps) => JSX.Element } = {
    [ObjectType.Organization]: OrganizationCreate,
    [ObjectType.Project]: ProjectCreate,
    [ObjectType.Routine]: RoutineCreate,
    [ObjectType.Standard]: StandardCreate,
}

/**
 * Maps object types to update components
 */
const updateMap: { [key in ObjectType]?: (props: UpdatePageProps) => JSX.Element } = {
    [ObjectType.Organization]: OrganizationUpdate,
    [ObjectType.Project]: ProjectUpdate,
    [ObjectType.Routine]: RoutineUpdate,
    [ObjectType.Standard]: StandardUpdate,
}

/**
 * Maps object types to view components
 */
const viewMap: { [key in ObjectType]?: (props: ViewPageProps) => JSX.Element } = {
    [ObjectType.Organization]: OrganizationView,
    [ObjectType.Project]: ProjectView,
    [ObjectType.Routine]: RoutineView,
    [ObjectType.Standard]: StandardView,
}


export const ObjectPage = ({
    session,
}: ObjectPageProps) => {
    const [location, setLocation] = useLocation();
    console.log('in object page', location)

    // Determine if page should be displayed as a dialog or full page. 
    // Also checks if the create, update, or view page should be shown
    const { isDialog, objectType, pageType, title } = useMemo(() => {
        console.log('calculating thingies', location.split('/')[1])
        const objectType = typeMap['/' + location.split('/')[1]];
        console.log('checking isdialog', sessionStorage);
        // Read session storage to check previous page.
        // Full page is only opened when 
        const isDialog = Boolean(sessionStorage.getItem('lastPath'));
        // Determine if create, update, or view page should be shown using the URL
        let pageType: PageType = PageType.View;
        if (location.endsWith('/add')) pageType = PageType.Create;
        else if (location.includes('/edit/')) pageType = PageType.Update;
        // Determine title for dialog
        const title = (objectType && objectType in titleMap) ? titleMap[objectType] : '';
        return { isDialog, objectType, pageType, title };
    }, [location]);

    const onAction = useCallback((action: ObjectDialogAction, item?: { id: string }) => {
        // Only navigate back if in a dialog (i.e. there is a previous page)
        const pageRoot = location.split('/')[1];
        switch (action) {
            case ObjectDialogAction.Add:
                setLocation(`${pageRoot}/${item?.id}`, { replace: !isDialog });
                break;
            case ObjectDialogAction.Cancel:
            case ObjectDialogAction.Close:
                if (isDialog) window.history.back();
                else setLocation(APP_LINKS.Home);
                break;
            case ObjectDialogAction.Edit:
                setLocation(`${pageRoot}/edit/${item?.id}`);
                break;
            case ObjectDialogAction.Save:
                if (isDialog) window.history.back();
                else setLocation(APP_LINKS.Home);
                break;
        }
    }, [isDialog, location, setLocation]);

    const displayedPage = useMemo<JSX.Element | undefined>(() => {
        console.log('calculating displayed page', pageType, objectType)
        if (!objectType) return undefined;
        if (pageType === PageType.Create) {
            const Create = createMap[objectType];
            document.title = `View ${titleMap[objectType]}`;
            return (Create && <Create
                onCancel={() => onAction(ObjectDialogAction.Cancel)}
                onCreated={(data: { id: string }) => onAction(ObjectDialogAction.Add, data)}
                session={session}
                zIndex={200}
            />)
        }
        if (pageType === PageType.Update) {
            const Update = updateMap[objectType];
            document.title = `Update ${titleMap[objectType]}`;
            return (Update && <Update
                onCancel={() => onAction(ObjectDialogAction.Cancel)}
                onUpdated={(data: { id: string }) => onAction(ObjectDialogAction.Save, data)}
                session={session}
                zIndex={200}
            />)
        }
        const View = viewMap[objectType];
        document.title = `View ${titleMap[objectType]}`;
        return View && <View session={session} zIndex={200} />
    }, [objectType, onAction, pageType, session]);

    return (
        <Box sx={{
            minHeight: '100vh',
            paddingTop: { xs: '64px', md: '80px' },
            paddingBottom: 'calc(56px + env(safe-area-inset-bottom))',
        }}>
            {/* If dialog not open, display full-page create, update, or view */}
            {!isDialog && displayedPage}
            {/* If dialog open, display dialog above view page */}
            {/* {isDalog && (
                <ProjectView session={session} zIndex={200} />
            )} */}
            {isDialog && displayedPage && (
                <BaseObjectDialog
                    onAction={onAction}
                    open={true}
                    title={`${pageType} ${title}`}
                    zIndex={200}
                >
                    {displayedPage}
                </BaseObjectDialog>
            )}
        </Box>
    )
}