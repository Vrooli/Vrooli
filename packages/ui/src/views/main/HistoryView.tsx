import { ListObject, ModelType, getObjectUrlBase } from "@local/shared";
import { IconButton, useTheme } from "@mui/material";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { PageTabs } from "../../components/PageTabs/PageTabs.js";
import { SideActionsButtons } from "../../components/buttons/SideActionsButtons/SideActionsButtons.js";
import { SearchList, SearchListScrollContainer } from "../../components/lists/SearchList/SearchList.js";
import { TopBar } from "../../components/navigation/TopBar.js";
import { useFindMany } from "../../hooks/useFindMany.js";
import { useTabs } from "../../hooks/useTabs.js";
import { IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { historyTabParams } from "../../utils/search/objectToSearch.js";
import { HistoryViewProps } from "./types.js";

const scrollContainerId = "history-search-scroll";

export function HistoryView({
    display,
    onClose,
}: HistoryViewProps) {
    const { t } = useTranslation();
    const { palette } = useTheme();
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
        <SearchListScrollContainer id={scrollContainerId}>
            <TopBar
                display={display}
                onClose={onClose}
                title={currTab.label}
                titleBehaviorDesktop="ShowIn"
                below={<PageTabs<typeof historyTabParams>
                    ariaLabel="history-tabs"
                    currTab={currTab}
                    fullWidth
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
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
    );
}
