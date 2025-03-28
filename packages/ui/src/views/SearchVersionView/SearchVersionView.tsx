import { LINKS, ListObject, ModelType, getObjectUrlBase } from "@local/shared";
import { useTheme } from "@mui/material";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { PageTabs } from "../../components/PageTabs/PageTabs.js";
import { SideActionsButtons } from "../../components/buttons/SideActionsButtons/SideActionsButtons.js";
import { SearchList, SearchListScrollContainer } from "../../components/lists/SearchList/SearchList.js";
import { TopBar } from "../../components/navigation/TopBar.js";
import { SessionContext } from "../../contexts.js";
import { useFindMany } from "../../hooks/useFindMany.js";
import { useTabs } from "../../hooks/useTabs.js";
import { IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { SideActionsButton } from "../../styles.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { scrollIntoFocusedView } from "../../utils/display/scroll.js";
import { PubSub } from "../../utils/pubsub.js";
import { searchVersionViewTabParams } from "../../utils/search/objectToSearch.js";
import { SearchVersionViewProps } from "../types.js";

const scrollContainerId = "version-search-scroll";

/**
 * Uncommon search page for versioned objects
 */
export function SearchVersionView({
    display,
    onClose,
}: SearchVersionViewProps) {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
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
        controlsUrl: display === "page",
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
        <SearchListScrollContainer id={scrollContainerId}>
            <TopBar
                display={display}
                onClose={onClose}
                title={t("SearchVersions")}
                titleBehaviorDesktop="ShowIn"
                below={<PageTabs<typeof searchVersionViewTabParams>
                    ariaLabel="search-version-tabs"
                    fullWidth
                    id="search-version-tabs"
                    ignoreIcons
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            {searchType && <SearchList
                {...findManyData}
                display={display}
                scrollContainerId={scrollContainerId}
            />}
            <SideActionsButtons display={display}>
                <SideActionsButton
                    aria-label={t("FilterList")}
                    onClick={focusSearch}
                >
                    <IconCommon
                        decorative
                        fill={palette.secondary.contrastText}
                        name="Search"
                        size={36}
                    />
                </SideActionsButton>
                {userId ? (
                    <SideActionsButton
                        aria-label={t("Add")}
                        onClick={onCreateStart}
                    >
                        <IconCommon
                            decorative
                            fill={palette.secondary.contrastText}
                            name="Add"
                        />
                    </SideActionsButton>
                ) : null}
            </SideActionsButtons>
        </SearchListScrollContainer>
    );
}
