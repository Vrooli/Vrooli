import { useCallback, useMemo } from "react";
import { Box } from "@mui/material"
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
    // Get URL params
    const [matchView, paramsView] = useRoute(`${APP_LINKS.Organization}/:id`); // View a specific organization
    const [matchUpdate, paramsUpdate] = useRoute(`${APP_LINKS.SearchOrganizations}/update/:id`);
    const id = useMemo(() => paramsView?.id ?? paramsUpdate?.id ?? '', [paramsView, paramsUpdate]);

    const isAddDialogOpen = useMemo(() => Boolean(matchView) && paramsView?.id === 'add', [matchView, paramsView]);
    const isEditDialogOpen = useMemo(() => Boolean(matchUpdate), [matchUpdate]);

    const onAction = useCallback((action: ObjectDialogAction, data?: any) => {
        switch (action) {
            case ObjectDialogAction.Add:
                if (data?.id) setLocation(`${APP_LINKS.Organization}/${data?.id}`, { replace: true });
                else setLocation(APP_LINKS.Organization, { replace: true });
                break;
            case ObjectDialogAction.Cancel:
                setLocation(APP_LINKS.Organization, { replace: true });
                break;
            case ObjectDialogAction.Close:
                setLocation(APP_LINKS.Organization, { replace: true });
                break;
            case ObjectDialogAction.Edit:
                setLocation(`${APP_LINKS.Organization}/update/${id}`, { replace: true });
                break;
            case ObjectDialogAction.Save:
                if (data?.id) setLocation(`${APP_LINKS.Organization}/${id}`, { replace: true });
                break;
        }
    }, [id, setLocation]);

    return (
        <Box pt="10vh" sx={{ minHeight: '88vh' }}>
            {/* Add dialog */}
            <BaseObjectDialog
                title={"Add Organization"}
                open={isAddDialogOpen}
                hasPrevious={false}
                hasNext={false}
                onAction={onAction}
            >
                <OrganizationCreate
                    session={session}
                    onCreated={(data: Organization) => onAction(ObjectDialogAction.Add, data)}
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                />
            </BaseObjectDialog>
            {/* Update dialog */}
            <BaseObjectDialog
                title={"Update Organization"}
                open={isEditDialogOpen}
                hasPrevious={false}
                hasNext={false}
                onAction={onAction}
            >
                <OrganizationUpdate
                    session={session}
                    onUpdated={() => onAction(ObjectDialogAction.Save)}
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                />
            </BaseObjectDialog>
            {/* Main view */}
            <OrganizationView session={session} />
        </Box>
    )
}