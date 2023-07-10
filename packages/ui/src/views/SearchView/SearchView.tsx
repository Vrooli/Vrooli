import { AddIcon, ApiIcon, CommonKey, GqlModelType, HelpIcon, LINKS, NoteIcon, OrganizationIcon, ProjectIcon, RoutineIcon, SmartContractIcon, StandardIcon, useLocation, UserIcon } from "@local/shared";
import { Box, Button, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { ShareSiteDialog } from "components/dialogs/ShareSiteDialog/ShareSiteDialog";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { centeredDiv } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { useTabs } from "utils/hooks/useTabs";
import { getObjectUrlBase } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { SearchPageTabOption, SearchType } from "utils/search/objectToSearch";
import { SessionContext } from "utils/SessionContext";
import { SearchViewProps } from "../types";

// Data for each tab
const tabParams = [{
    Icon: RoutineIcon,
    titleKey: "Routine" as CommonKey,
    popupTitleKey: "Add" as CommonKey,
    popupTooltipKey: "AddTooltip" as CommonKey,
    searchType: SearchType.Routine,
    tabType: SearchPageTabOption.Routines,
    where: { isInternal: false },
}, {
    Icon: ProjectIcon,
    titleKey: "Project" as CommonKey,
    popupTitleKey: "Add" as CommonKey,
    popupTooltipKey: "AddTooltip" as CommonKey,
    searchType: SearchType.Project,
    tabType: SearchPageTabOption.Projects,
    where: {},
}, {
    Icon: HelpIcon,
    titleKey: "Question" as CommonKey,
    popupTitleKey: "Invite" as CommonKey,
    popupTooltipKey: "AddTooltip" as CommonKey,
    searchType: SearchType.Question,
    tabType: SearchPageTabOption.Questions,
    where: {},
}, {
    Icon: NoteIcon,
    titleKey: "Note" as CommonKey,
    popupTitleKey: "Add" as CommonKey,
    popupTooltipKey: "AddTooltip" as CommonKey,
    searchType: SearchType.Note,
    tabType: SearchPageTabOption.Notes,
    where: {},
}, {
    Icon: OrganizationIcon,
    titleKey: "Organization" as CommonKey,
    popupTitleKey: "Add" as CommonKey,
    popupTooltipKey: "AddTooltip" as CommonKey,
    searchType: SearchType.Organization,
    tabType: SearchPageTabOption.Organizations,
    where: {},
}, {
    Icon: UserIcon,
    titleKey: "User" as CommonKey,
    popupTitleKey: "Invite" as CommonKey,
    popupTooltipKey: "InviteTooltip" as CommonKey,
    searchType: SearchType.User,
    tabType: SearchPageTabOption.Users,
    where: {},
}, {
    Icon: StandardIcon,
    titleKey: "Standard" as CommonKey,
    popupTitleKey: "Add" as CommonKey,
    popupTooltipKey: "AddTooltip" as CommonKey,
    searchType: SearchType.Standard,
    tabType: SearchPageTabOption.Standards,
    where: {},
}, {
    Icon: ApiIcon,
    titleKey: "Api" as CommonKey,
    popupTitleKey: "Add" as CommonKey,
    popupTooltipKey: "AddTooltip" as CommonKey,
    searchType: SearchType.Api,
    tabType: SearchPageTabOption.Apis,
    where: {},
}, {
    Icon: SmartContractIcon,
    titleKey: "SmartContract" as CommonKey,
    popupTitleKey: "Add" as CommonKey,
    popupTooltipKey: "AddTooltip" as CommonKey,
    searchType: SearchType.SmartContract,
    tabType: SearchPageTabOption.SmartContracts,
    where: {},
}];

/**
 * Search page for organizations, projects, routines, standards, users, and other main objects
 */
export const SearchView = ({
    display = "page",
    onClose,
    zIndex,
}: SearchViewProps) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Popup button, which opens either an add or invite dialog
    const [popupButton, setPopupButton] = useState<boolean>(false);

    const [shareDialogOpen, setShareDialogOpen] = useState(false);
    const closeShareDialog = useCallback(() => setShareDialogOpen(false), []);

    const {
        currTab,
        handleTabChange,
        popupTitleKey,
        popupTooltipKey,
        searchType,
        tabs,
        where,
    } = useTabs<SearchPageTabOption, { popupTitleKey: CommonKey, popupTooltipKey: CommonKey }>(tabParams, 0);

    // On tab change, update document title
    useEffect(() => {
        document.title = `${t("Search")} | ${currTab.label}`;
    }, [currTab, t]);

    const onAddClick = useCallback((ev: any) => {
        const addUrl = `${getObjectUrlBase({ __typename: searchType as `${GqlModelType}` })}/add`;
        // If not logged in, redirect to login page
        if (!getCurrentUser(session).id) {
            PubSub.get().publishSnack({ messageKey: "MustBeLoggedIn", severity: "Error" });
            setLocation(LINKS.Start, { searchParams: { redirect: addUrl } });
            return;
        }
        // If search type is a routine, open create routine page
        if (searchType === SearchType.Routine) {
            setLocation(`${LINKS.Routine}/add`);
        }
        // If search type is a user, open share dialog
        else if (searchType === SearchType.User) {
            setShareDialogOpen(true);
        }
        // Otherwise, navigate to add page
        else {
            setLocation(addUrl);
        }
    }, [searchType, session, setLocation]);

    const onPopupButtonClick = useCallback((ev: any) => {
        if ([SearchPageTabOption.Users].includes(currTab.value)) {
            setShareDialogOpen(true);
        } else {
            onAddClick(ev);
        }
    }, [currTab.value, onAddClick]);

    const handleScrolledFar = useCallback(() => { setPopupButton(true); }, []);
    const popupButtonContainer = useMemo(() => (
        <Box sx={{ ...centeredDiv, paddingTop: 1 }}>
            <Tooltip title={t(popupTooltipKey)}>
                <Button
                    onClick={onPopupButtonClick}
                    size="large"
                    variant="contained"
                    sx={{
                        zIndex: 100,
                        minWidth: "min(100%, 200px)",
                        height: "48px",
                        borderRadius: 3,
                        position: "fixed",
                        bottom: "calc(5em + env(safe-area-inset-bottom))",
                        transform: popupButton ? "translateY(0)" : "translateY(calc(10em + env(safe-area-inset-bottom)))",
                        transition: "transform 1s ease-in-out",
                    }}
                >
                    {t(popupTitleKey)}
                </Button>
            </Tooltip>
        </Box>
    ), [onPopupButtonClick, popupButton, popupTitleKey, popupTooltipKey, t]);

    return (
        <>
            <TopBar
                display={display}
                hideTitleOnDesktop={true}
                onClose={onClose}
                title={t("Search")}
                below={<PageTabs
                    ariaLabel="search-tabs"
                    id="search-tabs"
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
                zIndex={zIndex}
            />
            {/* Invite dialog for organizations and users */}
            <ShareSiteDialog
                onClose={closeShareDialog}
                open={shareDialogOpen}
                zIndex={zIndex + 2}
            />
            <Stack direction="row" alignItems="center" justifyContent="center" sx={{ paddingTop: 2 }}>
                <Typography component="h2" variant="h4">{t(searchType as CommonKey, { count: 2, defaultValue: searchType })}</Typography>
                <Tooltip title="Add new" placement="top">
                    <IconButton
                        size="medium"
                        onClick={onAddClick}
                        sx={{
                            padding: 1,
                        }}
                    >
                        <AddIcon fill={palette.secondary.main} width='1.5em' height='1.5em' />
                    </IconButton>
                </Tooltip>
            </Stack>
            {searchType && <SearchList
                id="main-search-page-list"
                dummyLength={display === "page" ? 5 : 3}
                take={20}
                searchType={searchType}
                onScrolledFar={handleScrolledFar}
                zIndex={zIndex}
                where={where}
            />}
            {popupButtonContainer}
        </>
    );
};
