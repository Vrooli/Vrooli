import { Chat, CommonKey, DeleteOneInput, DeleteType, endpointPostDeleteOne, endpointPutNotification, endpointPutNotificationsMarkAllAsRead, FindByIdInput, Notification, Success } from "@local/shared";
import { Tooltip, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { ChatListItemActions, NotificationListItemActions } from "components/lists/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { AddIcon, CommentIcon, CompleteIcon, NotificationsAllIcon } from "icons";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { openLink, useLocation } from "route";
import { ListObject } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { useDisplayServerError } from "utils/hooks/useDisplayServerError";
import { useFindMany } from "utils/hooks/useFindMany";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { useTabs } from "utils/hooks/useTabs";
import { openObject } from "utils/navigation/openObject";
import { InboxPageTabOption, SearchType } from "utils/search/objectToSearch";
import { ChatUpsert } from "views/objects/chat/ChatUpsert/ChatUpsert";
import { InboxViewProps } from "../types";

const tabParams = [{
    Icon: CommentIcon,
    titleKey: "Message" as CommonKey,
    searchType: SearchType.Chat,
    tabType: InboxPageTabOption.Messages,
    where: {},
}, {
    Icon: NotificationsAllIcon,
    titleKey: "Notification" as CommonKey,
    searchType: SearchType.Notification,
    tabType: InboxPageTabOption.Notifications,
    where: {},
}];

type InboxType = "Chat" | "Notification";
type InboxObject = Chat | Notification;

export const InboxView = ({
    isOpen,
    onClose,
    zIndex,
}: InboxViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const { palette } = useTheme();
    const display = toDisplay(isOpen);

    const { currTab, handleTabChange, searchType, tabs, title, where } = useTabs<InboxPageTabOption>(tabParams, 0);

    const {
        allData,
        loading,
        loadMore,
        setAllData,
    } = useFindMany<InboxObject>({
        searchType,
        where,
    });
    console.log("usetabs data", allData, currTab, searchType, where);

    const [deleteMutation, { errors: deleteErrors }] = useLazyFetch<DeleteOneInput, Success>(endpointPostDeleteOne);
    const [markAsReadMutation, { errors: markErrors }] = useLazyFetch<FindByIdInput, Success>(endpointPutNotification);
    const [markAllAsReadMutation, { errors: markAllErrors }] = useLazyFetch<undefined, Success>(endpointPutNotificationsMarkAllAsRead);
    useDisplayServerError(deleteErrors ?? markErrors ?? markAllErrors);

    const openNotification = useCallback((notification: Notification) => {
        if (notification.link) {
            openLink(setLocation, notification.link);
        }
    }, [setLocation]);

    const openChat = useCallback((chat: Chat) => {
        openObject(chat, setLocation);
    }, [setLocation]);

    const onDelete = useCallback((id: string, objectType: InboxType) => {
        fetchLazyWrapper<DeleteOneInput, Success>({
            fetch: deleteMutation,
            inputs: { id, objectType: objectType as DeleteType },
            successCondition: (data) => data.success,
            onSuccess: () => {
                setAllData(n => n.filter(n => n.id !== id));
            },
        });
    }, [deleteMutation, setAllData]);

    const onMarkAsRead = useCallback((id: string, objectType: InboxType) => {
        // TODO handle chats
        if (objectType === "Chat") return;
        fetchLazyWrapper<FindByIdInput, Success>({
            fetch: markAsReadMutation,
            inputs: { id },
            successCondition: (data) => data.success,
            onSuccess: () => {
                setAllData(n => n.map(n => n.id === id ? { ...n, isRead: true } : n));
            },
        });
    }, [markAsReadMutation, setAllData]);

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

    const [isCreateChatOpen, setIsCreateChatOpen] = useState(false);
    const openCreateChat = useCallback(() => { setIsCreateChatOpen(true); }, []);
    const closeCreateChat = useCallback(() => { setIsCreateChatOpen(false); }, []);
    const onChatCreated = useCallback((chat: Chat) => {
        closeCreateChat();
        //TODO
    }, [closeCreateChat]);

    const [onActionButtonPress, ActionButtonIcon, actionTooltip] = useMemo(() => {
        if (currTab.value === InboxPageTabOption.Notifications) {
            return [onMarkAllAsRead, CompleteIcon, "MarkAllAsRead"] as const;
        }
        return [openCreateChat, AddIcon, "CreateChat"] as const;
    }, [currTab.value, onMarkAllAsRead, openCreateChat]);

    const onAction = useCallback((action: keyof (ChatListItemActions | NotificationListItemActions), id: string) => {
        switch (action) {
            case "Delete":
                onDelete(id, searchType as InboxType);
                break;
            case "MarkAsRead":
                onMarkAsRead(id, searchType as InboxType);
                break;
        }
    }, [onDelete, onMarkAsRead, searchType]);

    const onClick = useCallback((item: InboxObject) => {
        switch (searchType) {
            case SearchType.Chat:
                openChat(item as Chat);
                break;
            case SearchType.Notification:
                openNotification(item as Notification);
                break;
        }
    }, [openChat, openNotification, searchType]);

    // If near the bottom of the page, load more data
    const handleScroll = useCallback(() => {
        const scrolledY = window.scrollY;
        const windowHeight = window.innerHeight;
        if (!loading && scrolledY > windowHeight - 500) {
            loadMore();
        }
    }, [loading, loadMore]);

    // Set event listener for infinite scroll
    useEffect(() => {
        window.addEventListener("scroll", handleScroll);
        return () => window.removeEventListener("scroll", handleScroll);
    }, [handleScroll]);


    return (
        <>
            {/* Create chat dialog */}
            <ChatUpsert
                isCreate={true}
                isOpen={isCreateChatOpen}
                onCancel={closeCreateChat}
                onCompleted={onChatCreated}
                overrideObject={{ __typename: "Chat" }}
                zIndex={zIndex + 1001}
            />
            {/* Main content */}
            <TopBar
                display={display}
                onClose={onClose}
                title={title}
                below={<PageTabs
                    ariaLabel="inbox-tabs"
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
                zIndex={zIndex}
            />
            <ListContainer
                emptyText={t("NoResults", { ns: "error" })}
                isEmpty={allData.length === 0 && !loading}
            >
                <ObjectList
                    dummyItems={new Array(5).fill(searchType)}
                    items={allData as ListObject[]}
                    keyPrefix={`${searchType}-list-item`}
                    loading={loading}
                    onAction={onAction}
                    onClick={(item) => onClick(item as InboxObject)}
                    zIndex={zIndex}
                />
            </ListContainer>
            {/* New Chat button */}
            <SideActionButtons
                display={display}
                zIndex={zIndex + 1}
                sx={{ position: "fixed" }}
            >
                <Tooltip title={t(actionTooltip)}>
                    <ColorIconButton aria-label="new-chat" background={palette.secondary.main} onClick={onActionButtonPress} >
                        <ActionButtonIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                </Tooltip>
            </SideActionButtons>
        </>
    );
};
