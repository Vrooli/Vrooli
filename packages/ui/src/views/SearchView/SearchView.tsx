import { AddIcon, ApiIcon, CommonKey, GqlModelType, HelpIcon, LINKS, NoteIcon, OrganizationIcon, ProjectIcon, RoutineIcon, SmartContractIcon, StandardIcon, useLocation, UserIcon } from "@local/shared";
import { Box, Button, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
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
    searchType: SearchType.Routine,
    tabType: SearchPageTabOption.Routines,
    where: { isInternal: false },
}, {
    Icon: ProjectIcon,
    titleKey: "Project" as CommonKey,
    searchType: SearchType.Project,
    tabType: SearchPageTabOption.Projects,
    where: {},
}, {
    Icon: HelpIcon,
    titleKey: "Question" as CommonKey,
    searchType: SearchType.Question,
    tabType: SearchPageTabOption.Questions,
    where: {},
}, {
    Icon: NoteIcon,
    titleKey: "Note" as CommonKey,
    searchType: SearchType.Note,
    tabType: SearchPageTabOption.Notes,
    where: {},
}, {
    Icon: OrganizationIcon,
    titleKey: "Organization" as CommonKey,
    searchType: SearchType.Organization,
    tabType: SearchPageTabOption.Organizations,
    where: {},
}, {
    Icon: UserIcon,
    titleKey: "User" as CommonKey,
    searchType: SearchType.User,
    tabType: SearchPageTabOption.Users,
    where: {},
}, {
    Icon: StandardIcon,
    titleKey: "Standard" as CommonKey,
    searchType: SearchType.Standard,
    tabType: SearchPageTabOption.Standards,
    where: {},
}, {
    Icon: ApiIcon,
    titleKey: "Api" as CommonKey,
    searchType: SearchType.Api,
    tabType: SearchPageTabOption.Apis,
    where: {},
}, {
    Icon: SmartContractIcon,
    titleKey: "SmartContract" as CommonKey,
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

    // Popup button for adding new objects
    const [popupButton, setPopupButton] = useState<boolean>(false);

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs<SearchPageTabOption>(tabParams, 0);

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
        // Otherwise, navigate to add page
        else {
            setLocation(addUrl);
        }
    }, [searchType, session, setLocation]);

    const handleScrolledFar = useCallback(() => { setPopupButton(true); }, []);
    const popupButtonContainer = useMemo(() => (
        <Box sx={{ ...centeredDiv, paddingTop: 1 }}>
            <Tooltip title={t("AddTooltip")}>
                <Button
                    onClick={onAddClick}
                    size="large"
                    variant="contained"
                    startIcon={<AddIcon />}
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
                    {t("Add")}
                </Button>
            </Tooltip>
        </Box>
    ), [onAddClick, popupButton, t]);

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
