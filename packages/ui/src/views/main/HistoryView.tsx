import { ListObject, ModelType, getObjectUrlBase } from "@local/shared";
import { IconButton } from "@mui/material";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../components/Page/Page.js";
import { PageTabs } from "../../components/PageTabs/PageTabs.js";
import { SideActionsButtons } from "../../components/buttons/SideActionsButtons.js";
import { SearchList, SearchListScrollContainer } from "../../components/lists/SearchList/SearchList.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { useFindMany } from "../../hooks/useFindMany.js";
import { useTabs } from "../../hooks/useTabs.js";
import { IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { ELEMENT_CLASSES } from "../../utils/consts.js";
import { historyTabParams } from "../../utils/search/objectToSearch.js";
import { HistoryViewProps } from "./types.js";

const scrollContainerId = "history-search-scroll";
const pageContainerStyle = {
    [`& .${ELEMENT_CLASSES.SearchBar}`]: {
        margin: 2,
    },
} as const;

export function HistoryView({
    display,
}: HistoryViewProps) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs({ id: "history-tabs", tabParams: historyTabParams, display });

    const findManyData = useFindMany<ListObject>({
        controlsUrl: display === "page",
        searchType,
        take: 20,
        where: where(undefined),
    });

    const handleAddBookmarkListClick = useCallback(function handleAddBookmarkListClickCallback() {
        setLocation(`${getObjectUrlBase({ __typename: ModelType.BookmarkList })}/add`);
    }, [setLocation]);

    return (
        <PageContainer size="fullSize" sx={pageContainerStyle}>
            <Navbar title={currTab.label} />
            <PageTabs<typeof historyTabParams>
                ariaLabel="history-tabs"
                currTab={currTab}
                fullWidth
                onChange={handleTabChange}
                tabs={tabs}
            />
            <SearchListScrollContainer id={scrollContainerId}>
                {
                    searchType && <SearchList
                        {...findManyData}
                        display={display}
                        scrollContainerId={scrollContainerId}
                    />
                }
                {
                    searchType === "BookmarkList" && <SideActionsButtons display={display}>
                        <IconButton
                            aria-label={t("Add")}
                            onClick={handleAddBookmarkListClick}
                        >
                            <IconCommon name="Add" />
                        </IconButton>
                    </SideActionsButtons>
                }
            </SearchListScrollContainer>
        </PageContainer>
    );
}
