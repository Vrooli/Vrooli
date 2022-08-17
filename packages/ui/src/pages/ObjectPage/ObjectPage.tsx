import { useCallback, useMemo } from "react";
import { Box } from "@mui/material"
import { BaseObjectDialog } from "components";
import { ObjectPageProps } from "../types";
import { ObjectDialogAction } from "components/dialogs/types";
import { useLocation } from '@shared/route';
import { APP_LINKS } from "@shared/consts";

enum PageType {
    Create = 'Create',
    Update = 'Update',
    View = 'View',
}

/**
 * Maps view page URLs to dialog titles
 */
const titleMap = {
    [APP_LINKS.Organization]: 'Organization',
    [APP_LINKS.Project]: 'Project',
    [APP_LINKS.Routine]: 'Routine',
    [APP_LINKS.Standard]: 'Standard',
}

export const ObjectPage = ({
    session,
    Create,
    Update,
    View,
}: ObjectPageProps) => {
    const [, setLocation] = useLocation();

    // Determine if page should be displayed as a dialog or full page. 
    // Also checks if the create, update, or view page should be shown
    const { isDialog, pageType, title } = useMemo(() => {
        console.log('checking isdialog', sessionStorage);
        // Read session storage to check previous page.
        // If previous page exists, assume this is a dialog.
        const isDialog = Boolean(sessionStorage.getItem('previousPage'));
        // Determine if create, update, or view page should be shown using the URL
        let pageType: PageType = PageType.View;
        const pathname = window.location.pathname;
        if (pathname.endsWith('/add')) pageType = PageType.Create;
        else if (pathname.includes('/edit/')) pageType = PageType.Update;
        // Determine title for dialog
        const objectLink = '/' + window.location.pathname.split('/')[1];
        const title = objectLink in titleMap ? titleMap[objectLink] : '';
        return { isDialog, pageType, title };
    }, []);

    const onAction = useCallback((action: ObjectDialogAction, item?: { id: string }) => {
        // Only navigate back if in a dialog (i.e. there is a previous page)
        const pageRoot = window.location.pathname.split('/')[1];
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
    }, [isDialog, setLocation]);

    const displayedPage = useMemo<JSX.Element>(() => {
        if (pageType === PageType.Create) return (<Create
            onCancel={() => onAction(ObjectDialogAction.Cancel)}
            onCreated={(data: { id: string }) => onAction(ObjectDialogAction.Add, data)}
            session={session}
            zIndex={200}
        />)
        if (pageType === PageType.Update) return (<Update
            onCancel={() => onAction(ObjectDialogAction.Cancel)}
            onUpdated={(data: { id: string }) => onAction(ObjectDialogAction.Save, data)}
            session={session}
            zIndex={200}
        />)
        return <View session={session} zIndex={200} />
    }, [Create, Update, View, onAction, pageType, session]);

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
            {isDialog && (
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