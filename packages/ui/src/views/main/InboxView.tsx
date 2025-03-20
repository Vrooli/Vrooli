import { Chat, endpointsNotification, getObjectUrlBase, InboxPageTabOption, ListObject, Notification, Success } from "@local/shared";
import { Tooltip, useTheme } from "@mui/material";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route/router.js";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { SideActionsButtons } from "../../components/buttons/SideActionsButtons/SideActionsButtons.js";
import { ListContainer } from "../../components/containers/ListContainer/ListContainer.js";
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
import { ActionIcon, AddIcon, CancelIcon, CompleteIcon, DeleteIcon } from "../../icons/common.js";
import { pagePaddingBottom, SideActionsButton } from "../../styles.js";
import { ArgsType } from "../../types.js";
import { BulkObjectAction } from "../../utils/actions/bulkObjectActions.js";
import { DUMMY_LIST_LENGTH } from "../../utils/consts.js";
import { inboxTabParams } from "../../utils/search/objectToSearch.js";
import { InboxViewProps } from "./types.js";

type InboxObject = Chat | Notification;

const scrollContainerId = "inbox-scroll-container";

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

    const [onActionButtonPress, ActionButtonIcon, actionTooltip] = useMemo(() => {
        if (currTab.key === InboxPageTabOption.Notification) {
            return [onMarkAllAsRead, CompleteIcon, "MarkAllAsRead"] as const;
        }
        return [openCreateChat, AddIcon, "CreateChat"] as const;
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

    useInfiniteScroll({
        loading,
        loadMore,
        scrollContainerId,
    });

    const actionIconProps = useMemo(() => ({ fill: palette.secondary.contrastText, width: "36px", height: "36px" }), [palette.secondary.contrastText]);

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
                    <SideActionsButton aria-label={t("Delete")} onClick={() => { onBulkActionStart(BulkObjectAction.Delete); }}>
                        <DeleteIcon {...actionIconProps} />
                    </SideActionsButton>
                </Tooltip> : null}
                <Tooltip title={t(isSelecting ? "Cancel" : "Select")}>
                    <SideActionsButton aria-label={t(isSelecting ? "Cancel" : "Select")} onClick={handleToggleSelecting}>
                        {isSelecting ? <CancelIcon {...actionIconProps} /> : <ActionIcon {...actionIconProps} />}
                    </SideActionsButton>
                </Tooltip>
                {!isSelecting ? <Tooltip title={t(actionTooltip)}>
                    <SideActionsButton aria-label={t(actionTooltip)} onClick={onActionButtonPress}>
                        <ActionButtonIcon {...actionIconProps} />
                    </SideActionsButton>
                </Tooltip> : null}
            </SideActionsButtons>
        </SearchListScrollContainer>
    );
}
