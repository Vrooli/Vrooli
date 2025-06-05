import { Box, IconButton, Typography, styled, useTheme } from "@mui/material";
import { BUSINESS_NAME, LINKS, getObjectUrlBase, type ListObject, type ModelType } from "@vrooli/shared";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../components/Page/Page.js";
import { PageTabs } from "../../components/PageTabs/PageTabs.js";
import { SideActionsButtons } from "../../components/buttons/SideActionsButtons.js";
import { SearchList, SearchListScrollContainer } from "../../components/lists/SearchList/SearchList.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { SessionContext } from "../../contexts/session.js";
import { useFindMany } from "../../hooks/useFindMany.js";
import { useTabs } from "../../hooks/useTabs.js";
import { useWindowSize } from "../../hooks/useWindowSize.js";
import { IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { ELEMENT_CLASSES, ELEMENT_IDS } from "../../utils/consts.js";
import { scrollIntoFocusedView } from "../../utils/display/scroll.js";
import { PubSub } from "../../utils/pubsub.js";
import { searchViewTabParams } from "../../utils/search/objectToSearch.js";
import { type SearchViewProps } from "../../views/types.js";

const scrollContainerId = "main-search-scroll";
const pageContainerStyle = {
    [`& .${ELEMENT_CLASSES.SearchBar}`]: {
        marginLeft: "auto",
        marginRight: "auto",
        maxWidth: "600px",
    },
} as const;

// Styled components for the search header
const SearchHeader = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    justifyContent: "center",
    padding: theme.spacing(3, 2),
    maxWidth: "600px",
    margin: "0 auto",
    textAlign: "center",
}));

const LogoContainer = styled(Box)(({ theme }) => ({
    display: "flex",
    alignItems: "center",
    marginBottom: theme.spacing(3),
    cursor: "pointer",
}));

const LogoName = styled(Typography)(({ theme }) => ({
    fontSize: "4em",
    fontFamily: "sakbunderan",
    lineHeight: "1.3",
    marginLeft: theme.spacing(1),
}));

const SearchBarContainer = styled(Box)(({ theme }) => ({
    width: "100%",
    maxWidth: "800px",
    margin: "0 auto",
}));

function LogoWithName({ onClick }: { onClick: () => void }) {
    return (
        <LogoContainer onClick={onClick}>
            <IconCommon
                decorative
                name="Vrooli"
                size={96}
            />
            <LogoName variant="h4">{BUSINESS_NAME}</LogoName>
        </LogoContainer>
    );
}

/**
 * Search page for teams, projects, routines, standards, users, and other main objects
 */
export function SearchView({
    display,
}: SearchViewProps) {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    // Track if header has been hidden at least once during this session
    const [headerHidden, setHeaderHidden] = useState(false);

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs({ id: ELEMENT_IDS.SearchTabs, tabParams: searchViewTabParams, display });

    // Reset headerHidden when tab changes
    useEffect(() => {
        setHeaderHidden(false);
    }, [searchType]);

    // Focus the search bar automatically when page loads or tab changes
    useEffect(() => {
        // Small delay to ensure the DOM is ready
        const timer = setTimeout(() => {
            const searchBarInput = document.querySelector(`#${scrollContainerId}-search-bar input`) as HTMLElement;
            if (searchBarInput) {
                searchBarInput.focus();
            }
        }, 100);
        return () => clearTimeout(timer);
    }, [searchType]);

    const findManyData = useFindMany<ListObject>({
        controlsUrl: display === "Page",
        searchType,
        take: 20,
        where: where(undefined),
    });

    const onCreateStart = useCallback(function onCreateStartCallback() {
        // If tab is 'All', go to "Create" page
        if (searchType === "Popular") {
            setLocation(LINKS.Create);
            return;
        }
        const addUrl = `${getObjectUrlBase({ __typename: searchType as `${ModelType}` })}/add`;
        // If not logged in, redirect to login page
        if (!userId) {
            PubSub.get().publish("snack", { messageKey: "NotLoggedIn", severity: "Error" });
            setLocation(LINKS.Login, { searchParams: { redirect: addUrl } });
            return;
        }
        // Otherwise, navigate to object's add page
        else setLocation(addUrl);
    }, [searchType, setLocation, userId]);

    function focusSearch() {
        const searchBarInput = document.querySelector(`#${scrollContainerId}-search-bar input`) as HTMLElement;
        if (searchBarInput) {
            scrollIntoFocusedView(scrollContainerId);
            searchBarInput.focus();
        }
    }

    const handleLogoClick = useCallback(() => {
        setLocation(LINKS.Home);
    }, [setLocation]);

    // Handler for search bar focus/blur
    const handleSearchFocus = useCallback(() => {
        if (isMobile) {
            setHeaderHidden(true);
        }
    }, [isMobile]);

    // No need to handle blur events for header visibility
    const handleSearchBlur = useCallback(() => {
        // Do nothing - we want to keep the header hidden
    }, []);

    // Dynamic header text based on the current tab
    const headerText = useMemo(() => {
        switch (searchType) {
            case "Popular":
                return {
                    primary: "DiscoverAIAgents" as const,
                    secondary: "SuperchargeWorkflow" as const,
                };
            case "Routine":
                return {
                    primary: "FindPerfectRoutine" as const,
                    secondary: "AutomateWorkWithRoutines" as const,
                };
            case "Project":
                return {
                    primary: "ExploreProjects" as const,
                    secondary: "FindCollaborativeSpaces" as const,
                };
            case "Team":
                return {
                    primary: "DiscoverTeams" as const,
                    secondary: "JoinForcesWithAgents" as const,
                };
            case "User":
                return {
                    primary: "FindUsersAndAgents" as const,
                    secondary: "ConnectWithPeople" as const,
                };
            case "Standard":
                return {
                    primary: "BrowseStandards" as const,
                    secondary: "FindDataStructure" as const,
                };
            case "Api":
                return {
                    primary: "ExploreAPIs" as const,
                    secondary: "ConnectWorkflows" as const,
                };
            case "Code":
                return {
                    primary: "DiscoverCodeComponents" as const,
                    secondary: "FindReusableCode" as const,
                };
            default:
                return {
                    primary: "SearchResources" as const,
                    secondary: "FindWhatYouNeed" as const,
                };
        }
    }, [searchType]);

    // Get placeholder text based on the current tab
    const searchPlaceholder = useMemo(() => {
        switch (searchType) {
            case "Popular":
                return "SearchForRoutinesTeamsUsers" as const;
            case "Routine":
                return "SearchForRoutines" as const;
            case "Project":
                return "SearchForProjects" as const;
            case "Team":
                return "SearchForTeams" as const;
            case "User":
                return "SearchForUsersAndAgents" as const;
            case "Standard":
                return "SearchForStandards" as const;
            case "Api":
                return "SearchForAPIs" as const;
            case "Code":
                return "SearchForCodeComponents" as const;
            default:
                return "Search" as const;
        }
    }, [searchType]);

    // Determine if we should show the header - show only when NOT on mobile or header has never been hidden
    const showHeader = useMemo(() => {
        return !isMobile || !headerHidden;
    }, [isMobile, headerHidden]);

    return (
        <PageContainer size="fullSize" sx={pageContainerStyle}>
            <Navbar title={t("Search")} titleBehavior="Hide" />
            <PageTabs<typeof searchViewTabParams>
                ariaLabel="Search tabs"
                fullWidth
                id={ELEMENT_IDS.SearchTabs}
                ignoreIcons
                currTab={currTab}
                onChange={handleTabChange}
                tabs={tabs}
            />
            <SearchListScrollContainer id={scrollContainerId}>
                {showHeader && (
                    <SearchHeader>
                        <LogoWithName onClick={handleLogoClick} />
                        <Typography variant="h5" color="text.primary" mb={1}>{t(headerText.primary)}</Typography>
                        <Typography variant="subtitle1" color="text.secondary" mb={3}>{t(headerText.secondary)}</Typography>
                    </SearchHeader>
                )}
                <SearchBarContainer>
                    {searchType && <SearchList
                        {...findManyData}
                        display={display}
                        scrollContainerId={scrollContainerId}
                        searchBarVariant="paper"
                        searchPlaceholder={t(searchPlaceholder)}
                        onSearchFocus={handleSearchFocus}
                        onSearchBlur={handleSearchBlur}
                    />}
                </SearchBarContainer>
            </SearchListScrollContainer>
            <SideActionsButtons display={display}>
                <IconButton
                    aria-label={t("FilterList")}
                    onClick={focusSearch}
                >
                    <IconCommon name="Search" />
                </IconButton>
                {userId ? (
                    <IconButton
                        aria-label={t("Add")}
                        onClick={onCreateStart}
                    >
                        <IconCommon name="Add" />
                    </IconButton>
                ) : null}
            </SideActionsButtons>
        </PageContainer>
    );
}
