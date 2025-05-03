import { Chat, endpointsNotification, getObjectUrlBase, InboxPageTabOption, ListObject, Notification, Success } from "@local/shared";
import { IconButton, Tooltip, useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { SideActionsButtons } from "../../components/buttons/SideActionsButtons.js";
import { ListContainer } from "../../components/containers/ListContainer.js";
import { ObjectList } from "../../components/lists/ObjectList/ObjectList.js";
import { SearchListScrollContainer } from "../../components/lists/SearchList/SearchList.js";
import { ObjectListActions } from "../../components/lists/types.js";
import { NavbarInner, NavListBox, NavListProfileButton, SiteNavigatorButton, TitleDisplay } from "../../components/navigation/Navbar.js";
import { PageContainer } from "../../components/Page/Page.js";
import { PageTabs } from "../../components/PageTabs/PageTabs.js";
import { useBulkObjectActions } from "../../hooks/objectActions.js";
import { useIsLeftHanded } from "../../hooks/subscriptions.js";
import { useLazyFetch } from "../../hooks/useFetch.js";
import { useSelectableList } from "../../hooks/useSelectableList.js";
import { useTabs } from "../../hooks/useTabs.js";
import { Icon, IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { useChats, useChatsStore } from "../../stores/chatsStore.js";
import { useNotifications, useNotificationsStore } from "../../stores/notificationsStore.js";
import { pagePaddingBottom } from "../../styles.js";
import { ViewDisplayType } from "../../types.js";
import { BulkObjectAction } from "../../utils/actions/bulkObjectActions.js";
import { DUMMY_LIST_LENGTH } from "../../utils/consts.js";
import { inboxTabParams } from "../../utils/search/objectToSearch.js";
import { InboxViewProps } from "./types.js";

type InboxObject = Chat | Notification;

const scrollContainerId = "inbox-scroll-container";
const cancelIconInfo = { name: "Cancel", type: "Common" } as const;
const addIconInfo = { name: "Add", type: "Common" } as const;
const selectIconInfo = { name: "Action", type: "Common" } as const;
const listContainerStyle = { marginBottom: pagePaddingBottom } as const;
const MARK_READ_TIMEOUT_MS = 2000;

export function InboxView({
    display,
}: InboxViewProps) {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const isLeftHanded = useIsLeftHanded();
    useNotifications();
    const storeNotifications = useNotificationsStore(state => state.notifications);
    const isLoadingStoreNotifications = useNotificationsStore(state => state.isLoading);
    const markAllStoreNotificationsAsRead = useNotificationsStore(state => state.markAllNotificationsAsRead);
    const markReadTimeoutRef = useRef<NodeJS.Timeout | null>(null);

    useChats();
    const storeChats = useChatsStore(state => state.chats);
    const isLoadingStoreChats = useChatsStore(state => state.isLoading);
    const removeChatFromStore = useChatsStore(state => state.removeChat);
    const updateChatInStore = useChatsStore(state => state.updateChat);

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs({ id: "inbox-tabs", tabParams: inboxTabParams, display: display as ViewDisplayType });

    const isNotificationTab = currTab.key === InboxPageTabOption.Notification;
    const isChatTab = currTab.key === InboxPageTabOption.Message;

    const currentRawItems = useMemo(() =>
        isNotificationTab ? storeNotifications : isChatTab ? storeChats : [],
        [isNotificationTab, isChatTab, storeNotifications, storeChats],
    );

    const isLoadingCurrent = isNotificationTab ? isLoadingStoreNotifications : isChatTab ? isLoadingStoreChats : false;

    const {
        isSelecting,
        handleToggleSelecting,
        handleToggleSelect,
        selectedData,
        setIsSelecting,
        setSelectedData,
    } = useSelectableList<InboxObject>(currentRawItems);
    const { onBulkActionStart, BulkDeleteDialogComponent } = useBulkObjectActions<InboxObject>({
        allData: currentRawItems,
        selectedData,
        setAllData: (newDataOrFn) => {
            if (isChatTab) {
                if (typeof newDataOrFn !== "function") {
                    const newData = newDataOrFn as Chat[];
                    const originalIds = new Set(storeChats.map(c => c.id));
                    const newIds = new Set(newData.map(c => c.id));
                    const deletedIds = [...originalIds].filter(id => !newIds.has(id));

                    deletedIds.forEach(id => removeChatFromStore(id));
                } else {
                    console.warn("InboxView: setAllData received a function; cannot update chat store accurately after delete. Consider refetching or modifying useBulkObjectActions.");
                }
            } else if (isNotificationTab) {
                console.warn("InboxView: setAllData called for notifications tab, store update not implemented.");
            }
        },
        setSelectedData: (data) => {
            setSelectedData(data);
            setIsSelecting(false);
        },
        setLocation,
    });

    const [markAllAsReadMutation] = useLazyFetch<undefined, Success>(endpointsNotification.markAllAsRead);

    const currentItems = useMemo(() =>
        currentRawItems as ListObject[],
        [currentRawItems],
    );

    const isEmptyCurrent = currentItems.length === 0 && !isLoadingCurrent;

    useEffect(() => {
        if (markReadTimeoutRef.current) {
            clearTimeout(markReadTimeoutRef.current);
            markReadTimeoutRef.current = null;
        }

        if (currTab.key === InboxPageTabOption.Notification) {
            const hasUnread = storeNotifications.some(item => !item.isRead);

            if (hasUnread) {
                markReadTimeoutRef.current = setTimeout(() => {
                    fetchLazyWrapper<undefined, Success>({
                        fetch: markAllAsReadMutation,
                        successCondition: (data) => data.success,
                        onSuccess: () => {
                            markAllStoreNotificationsAsRead();
                        },
                        onError: (error) => {
                            console.error("InboxView: markAllAsReadMutation failed:", error);
                        },
                    });
                }, MARK_READ_TIMEOUT_MS);
            }
        }

        return () => {
            if (markReadTimeoutRef.current) {
                clearTimeout(markReadTimeoutRef.current);
            }
        };
    }, [currTab, storeNotifications, markAllAsReadMutation, markAllStoreNotificationsAsRead]);

    const openCreateChat = useCallback(() => {
        setLocation(`${getObjectUrlBase({ __typename: "Chat" })}/add`);
    }, [setLocation]);

    const [onActionButtonPress, actionButtonIconInfo, actionTooltip] = useMemo(() => {
        if (isChatTab) {
            return [openCreateChat, addIconInfo, "CreateChat"] as const;
        }
        return [undefined, undefined, undefined] as const;
    }, [isChatTab, openCreateChat]);

    const onAction = useCallback((action: keyof ObjectListActions<InboxObject>, ...data: unknown[]) => {
        const item = data[0] as InboxObject;
        if (!item) return;

        const itemId = typeof item === "string" ? item : item.id;

        switch (action) {
            case "Deleted":
                if (isChatTab && itemId) {
                    removeChatFromStore(itemId);
                } else if (isNotificationTab) {
                    console.warn("Notification deletion from list not implemented via store yet.");
                }
                break;
            case "Updated":
                if (isChatTab && typeof item !== "string") {
                    updateChatInStore(item as Chat);
                } else if (isNotificationTab && typeof item !== "string") {
                    console.warn("Notification update from list not implemented via store yet.");
                }
                break;
        }
    }, [isChatTab, isNotificationTab, removeChatFromStore, updateChatInStore]);

    function handleDelete() {
        onBulkActionStart(BulkObjectAction.Delete);
    }

    return (
        <PageContainer size="fullSize">
            {BulkDeleteDialogComponent}
            <NavbarInner>
                <SiteNavigatorButton />
                <TitleDisplay title={currTab.label} />
                <NavListBox isLeftHanded={isLeftHanded}>
                    <NavListProfileButton />
                </NavListBox>
            </NavbarInner>
            <PageTabs<typeof inboxTabParams>
                ariaLabel="inbox-tabs"
                fullWidth
                id="inbox-tabs"
                ignoreIcons
                currTab={currTab}
                onChange={handleTabChange}
                tabs={tabs}
            />
            <SearchListScrollContainer id={scrollContainerId}>
                <ListContainer
                    emptyText={t("NoResults", { ns: "error" })}
                    isEmpty={isEmptyCurrent}
                    sx={listContainerStyle}
                >
                    <ObjectList
                        dummyItems={new Array(DUMMY_LIST_LENGTH).fill(isNotificationTab ? "Notification" : isChatTab ? "Chat" : undefined)}
                        handleToggleSelect={handleToggleSelect}
                        isSelecting={isSelecting}
                        items={currentItems}
                        keyPrefix={`${searchType}-list-item`}
                        loading={isLoadingCurrent}
                        onAction={onAction}
                        selectedItems={selectedData}
                    />
                </ListContainer>
            </SearchListScrollContainer>
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
                        <Icon info={isSelecting ? cancelIconInfo : selectIconInfo} />
                    </IconButton>
                </Tooltip>
                {!isSelecting && isChatTab && actionButtonIconInfo && actionTooltip ? (
                    <Tooltip title={t(actionTooltip)}>
                        <IconButton
                            aria-label={t(actionTooltip)}
                            onClick={onActionButtonPress}
                        >
                            <Icon info={actionButtonIconInfo} />
                        </IconButton>
                    </Tooltip>
                ) : null}
            </SideActionsButtons>
        </PageContainer>
    );
}
