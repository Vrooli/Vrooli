import { useCallback, useMemo } from "react";
import { Box, useTheme } from "@mui/material"
import { BaseObjectDialog, OrganizationView } from "components";
import { OrganizationViewPageProps } from "./types";
import { ObjectDialogAction } from "components/dialogs/types";
import { useLocation, useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { OrganizationCreate } from "components/views/OrganizationCreate/OrganizationCreate";
import { Organization } from "types";
import { OrganizationUpdate } from "components/views/OrganizationUpdate/OrganizationUpdate";

export const OrganizationViewPage = ({
    session
}: OrganizationViewPageProps) => {
    const [, setLocation] = useLocation();
    const { breakpoints } = useTheme();
    // Get URL params
    const [matchView, paramsView] = useRoute(`${APP_LINKS.Organization}/:id`);
    const [matchUpdate, paramsUpdate] = useRoute(`${APP_LINKS.Organization}/edit/:id`);
    const id = useMemo(() => paramsView?.id ?? paramsUpdate?.id ?? '', [paramsView, paramsUpdate]);

    const isAddDialogOpen = useMemo(() => Boolean(matchView) && paramsView?.id === 'add', [matchView, paramsView]);
    const isEditDialogOpen = useMemo(() => Boolean(matchUpdate), [matchUpdate]);

    const onAction = useCallback((action: ObjectDialogAction, data?: any) => {
        switch (action) {
            case ObjectDialogAction.Add:
                setLocation(`${APP_LINKS.Organization}/${data?.id}`, { replace: true });
                break;
            case ObjectDialogAction.Cancel:
                window.history.back();
                break;
            case ObjectDialogAction.Close:
                window.history.back();
                break;
            case ObjectDialogAction.Edit:
                setLocation(`${APP_LINKS.Organization}/edit/${id}`);
                break;
            case ObjectDialogAction.Save:
                window.history.back();
                break;
        }
    }, [id, setLocation]);

    return (
        <Box sx={{
            minHeight: '100vh',
            paddingTop: '64px',
            [breakpoints.up('md')]: {
                paddingTop: '8vh',
                minHeight: '88vh',
            },
        }}>
            {/* Add dialog */}
            <BaseObjectDialog
                hasNext={false}
                hasPrevious={false}
                onAction={onAction}
                open={isAddDialogOpen}
                title={"Add Organization"}
            >
                <OrganizationCreate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onCreated={(data: Organization) => onAction(ObjectDialogAction.Add, data)}
                    session={session}
                />
            </BaseObjectDialog>
            {/* Update dialog */}
            <BaseObjectDialog
                hasNext={false}
                hasPrevious={false}
                onAction={onAction}
                open={isEditDialogOpen}
                title={"Update Organization"}
            >
                <OrganizationUpdate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onUpdated={() => onAction(ObjectDialogAction.Save)}
                    session={session}
                />
            </BaseObjectDialog>
            {/* Main view */}
            <OrganizationView
                session={session}
            />
        </Box>
    )
}