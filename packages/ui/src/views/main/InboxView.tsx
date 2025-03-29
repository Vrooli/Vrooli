import { Chat, endpointsNotification, getObjectUrlBase, InboxPageTabOption, ListObject, Notification, Success } from "@local/shared";
import { IconButton, Tooltip, useTheme } from "@mui/material";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { SideActionsButtons } from "../../components/buttons/SideActionsButtons/SideActionsButtons.js";
import { ListContainer } from "../../components/containers/ListContainer.js";
import { ObjectList } from "../../components/lists/ObjectList/ObjectList.js";
import { SearchListScrollContainer } from "../../components/lists/SearchList/SearchList.js";
import { ObjectListActions } from "../../components/lists/types.js";
import { TopBar } from "../../components/navigation/TopBar.js";
import { PageTabs } from "../../components/PageTabs/PageTabs.js";
import { useInfiniteScroll } from "../../hooks/gestures.js";
import { useBulkObjectActions } from "../../hooks/objectActions.js";
import { useFindMany } from "../../hooks/useFindMany.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { useSelectableList } from "../../hooks/useSelectableList.js";
import { useTabs } from "../../hooks/useTabs.js";
import { Icon, IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { pagePaddingBottom } from "../../styles.js";
import { ArgsType } from "../../types.js";
import { BulkObjectAction } from "../../utils/actions/bulkObjectActions.js";
import { DUMMY_LIST_LENGTH } from "../../utils/consts.js";
import { inboxTabParams } from "../../utils/search/objectToSearch.js";
import { InboxViewProps } from "./types.js";

type InboxObject = Chat | Notification;

const scrollContainerId = "inbox-scroll-container";
const cancelIconInfo = { name: "Cancel", type: "Common" } as const;

export function InboxView({
    display,
    onClose,
}: InboxViewProps) {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs({ id: "inbox-tabs", tabParams: inboxTabParams, display });

    const {
        allData,
        loading,
        removeItem,
        loadMore,
        setAllData,
        updateItem,
    } = useFindMany<InboxObject>({
        searchType,
        where: where(undefined),
    });

    const {
        isSelecting,
        handleToggleSelecting,
        handleToggleSelect,
        selectedData,
        setIsSelecting,
        setSelectedData,
    } = useSelectableList<InboxObject>(allData);
    const { onBulkActionStart, BulkDeleteDialogComponent } = useBulkObjectActions<InboxObject>({
        allData,
        selectedData,
        setAllData,
        setSelectedData: (data) => {
            setSelectedData(data);
            setIsSelecting(false);
        },
        setLocation,
    });

    const [markAllAsReadMutation] = useLazyFetch<undefined, Success>(endpointsNotification.markAllAsRead);

    const onMarkAllAsRead = useCallback(() => {
        // TODO handle chats
        fetchLazyWrapper<any, Success>({
            fetch: markAllAsReadMutation,
            successCondition: (data) => data.success,
            onSuccess: () => {
                setAllData(n => n.map(n => ({ ...n, isRead: true })));
            },
        });
    }, [markAllAsReadMutation, setAllData]);

    const openCreateChat = useCallback(() => {
        setLocation(`${getObjectUrlBase({ __typename: "Chat" })}/add`);
    }, [setLocation]);

    const [onActionButtonPress, actionButtonIconInfo, actionTooltip] = useMemo(() => {
        if (currTab.key === InboxPageTabOption.Notification) {
            return [onMarkAllAsRead, { name: "Complete", type: "Common" }, "MarkAllAsRead"] as const;
        }
        return [openCreateChat, { name: "Add", type: "Common" }, "CreateChat"] as const;
    }, [currTab.key, onMarkAllAsRead, openCreateChat]);

    const onAction = useCallback((action: keyof ObjectListActions<InboxObject>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted":
                removeItem(...(data as ArgsType<ObjectListActions<InboxObject>["Deleted"]>));
                break;
            case "Updated":
                updateItem(...(data as ArgsType<ObjectListActions<InboxObject>["Updated"]>));
                break;
        }
    }, [removeItem, updateItem]);

    function handleDelete() {
        onBulkActionStart(BulkObjectAction.Delete);
    }

    useInfiniteScroll({
        loading,
        loadMore,
        scrollContainerId,
    });

    return (
        <SearchListScrollContainer id={scrollContainerId}>
            {BulkDeleteDialogComponent}
            <TopBar
                display={display}
                onClose={onClose}
                title={currTab.label}
                titleBehaviorDesktop="ShowIn"
                below={<PageTabs<typeof inboxTabParams>
                    ariaLabel="inbox-tabs"
                    fullWidth
                    id="inbox-tabs"
                    ignoreIcons
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            <ListContainer
                emptyText={t("NoResults", { ns: "error" })}
                isEmpty={allData.length === 0 && !loading}
                sx={{ marginBottom: pagePaddingBottom }}
            >
                <ObjectList
                    dummyItems={new Array(DUMMY_LIST_LENGTH).fill(searchType)}
                    handleToggleSelect={handleToggleSelect}
                    isSelecting={isSelecting}
                    items={allData as ListObject[]}
                    keyPrefix={`${searchType}-list-item`}
                    loading={loading}
                    onAction={onAction}
                    selectedItems={selectedData}
                />
            </ListContainer>
            <SideActionsButtons display={display}>
                {isSelecting && selectedData.length > 0 ? <Tooltip title={t("Delete")}>
                    <IconButton
                        aria-label={t("Delete")}
                        onClick={handleDelete}
                    >
                        <IconCommon
                            decorative
                            fill={palette.secondary.contrastText}
                            name="Delete"
                            size={36}
                        />
                    </IconButton>
                </Tooltip> : null}
                <Tooltip title={t(isSelecting ? "Cancel" : "Select")}>
                    <IconButton
                        aria-label={t(isSelecting ? "Cancel" : "Select")}
                        onClick={handleToggleSelecting}
                    >
                        <Icon info={isSelecting ? cancelIconInfo : actionButtonIconInfo} />
                    </IconButton>
                </Tooltip>
                {!isSelecting ? <Tooltip title={t(actionTooltip)}>
                    <IconButton
                        aria-label={t(actionTooltip)}
                        onClick={onActionButtonPress}
                    >
                        <Icon info={actionButtonIconInfo} />
                    </IconButton>
                </Tooltip> : null}
            </SideActionsButtons>
        </SearchListScrollContainer>
    );
}
