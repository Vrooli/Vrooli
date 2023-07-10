import { BookmarkFilledIcon, CommonKey, RoutineActiveIcon, RoutineCompleteIcon, RunStatus, VisibleIcon } from "@local/shared";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useTabs } from "utils/hooks/useTabs";
import { HistoryPageTabOption, SearchType } from "utils/search/objectToSearch";
import { HistoryViewProps } from "../types";

const tabParams = [{
    Icon: VisibleIcon,
    titleKey: "View" as CommonKey,
    searchType: SearchType.View,
    tabType: HistoryPageTabOption.Viewed,
    where: {},
}, {
    Icon: BookmarkFilledIcon,
    titleKey: "Bookmark" as CommonKey,
    searchType: SearchType.BookmarkList,
    tabType: HistoryPageTabOption.Bookmarked,
    where: {},
}, {
    Icon: RoutineActiveIcon,
    titleKey: "Active" as CommonKey,
    searchType: SearchType.RunProjectOrRunRoutine,
    tabType: HistoryPageTabOption.RunsActive,
    where: { statuses: [RunStatus.InProgress, RunStatus.Scheduled] },
}, {
    Icon: RoutineCompleteIcon,
    titleKey: "Complete" as CommonKey,
    searchType: SearchType.RunProjectOrRunRoutine,
    tabType: HistoryPageTabOption.RunsCompleted,
    where: { statuses: [RunStatus.Cancelled, RunStatus.Completed, RunStatus.Failed] },
}];

/**
 * Shows items you've bookmarked, viewed, or run recently.
 */
export const HistoryView = ({
    display = "page",
    onClose,
    zIndex,
}: HistoryViewProps) => {
    const { currTab, handleTabChange, searchType, tabs, title, where } = useTabs<HistoryPageTabOption>(tabParams, 0);

    return (
        <>
            <TopBar
                display={display}
                hideTitleOnDesktop={true}
                onClose={onClose}
                title={title}
                below={<PageTabs
                    ariaLabel="history-tabs"
                    currTab={currTab}
                    fullWidth
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
                zIndex={zIndex}
            />
            {searchType && <SearchList
                id="history-page-list"
                dummyLength={display === "page" ? 5 : 3}
                take={20}
                searchType={searchType}
                zIndex={zIndex}
                sxs={{
                    search: {
                        marginTop: 2,
                    },
                }}
                where={where}
            />}
        </>
    );
};
