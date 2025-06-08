import IconButton from "@mui/material/IconButton";
import { LINKS, getObjectUrlBase, type ListObject, type ModelType } from "@vrooli/shared";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../components/Page/Page.js";
import { PageTabs } from "../../components/PageTabs/PageTabs.js";
import { SideActionsButtons } from "../../components/buttons/SideActionsButtons.js";
import { SearchList, SearchListScrollContainer } from "../../components/lists/SearchList/SearchList.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { SessionContext } from "../../contexts/session.js";
import { useFindMany } from "../../hooks/useFindMany.js";
import { useTabs } from "../../hooks/useTabs.js";
import { IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { ELEMENT_CLASSES } from "../../utils/consts.js";
import { scrollIntoFocusedView } from "../../utils/display/scroll.js";
import { PubSub } from "../../utils/pubsub.js";
import { searchVersionViewTabParams } from "../../utils/search/objectToSearch.js";
import { type SearchVersionViewProps } from "../types.js";

const scrollContainerId = "version-search-scroll";
const pageContainerStyle = {
    [`& .${ELEMENT_CLASSES.SearchBar}`]: {
        margin: 2,
    },
} as const;

/**
 * Uncommon search page for versioned objects
 */
export function SearchVersionView({
    display,
}: SearchVersionViewProps) {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs({ id: "search-version-tabs", tabParams: searchVersionViewTabParams, display });

    const findManyData = useFindMany<ListObject>({
        controlsUrl: display === "Page",
        searchType,
        take: 20,
        where: where(undefined),
    });

    const onCreateStart = useCallback(function onCreateStartCallback(e: React.MouseEvent<HTMLElement>) {
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

    function focusSearch() { scrollIntoFocusedView("search-bar-version-search-page-list"); }

    return (
        <PageContainer size="fullSize" sx={pageContainerStyle}>
            <Navbar title={t("SearchVersions")} />
            <PageTabs<typeof searchVersionViewTabParams>
                ariaLabel="search-version-tabs"
                fullWidth
                id="search-version-tabs"
                ignoreIcons
                currTab={currTab}
                onChange={handleTabChange}
                tabs={tabs}
            />
            <SearchListScrollContainer id={scrollContainerId}>
                {searchType && <SearchList
                    {...findManyData}
                    display={display}
                    scrollContainerId={scrollContainerId}
                />}
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
            </SearchListScrollContainer>
        </PageContainer>
    );
}
