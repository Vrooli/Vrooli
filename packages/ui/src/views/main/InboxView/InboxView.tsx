import { AddIcon, Chat, CommentIcon, CommonKey, DeleteOneInput, deleteOneOrManyDeleteOne, DeleteType, FindByIdInput, Notification, notificationMarkAllAsRead, notificationMarkAsRead, NotificationsAllIcon, openLink, Success, useLocation } from "@local/shared";
import { Button, useTheme } from "@mui/material";
import { mutationWrapper, useCustomMutation } from "api";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { ChatListItemActions, NotificationListItemActions } from "components/lists/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useCallback, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { listToListItems } from "utils/display/listTools";
import { useDisplayApolloError } from "utils/hooks/useDisplayApolloError";
import { useFindMany } from "utils/hooks/useFindMany";
import { useTabs } from "utils/hooks/useTabs";
import { openObject } from "utils/navigation/openObject";
import { InboxPageTabOption, SearchType } from "utils/search/objectToSearch";
import { InboxViewProps } from "../types";

const tabParams = [{
    Icon: NotificationsAllIcon,
    titleKey: "Notification" as CommonKey,
    searchType: SearchType.Notification,
    tabType: InboxPageTabOption.Notifications,
    where: {},
}, {
    Icon: CommentIcon,
    titleKey: "Message" as CommonKey,
    searchType: SearchType.Chat,
    tabType: InboxPageTabOption.Messages,
    where: {},
}];

type InboxType = "Chat" | "Notification";
type InboxObject = Chat | Notification;

export const InboxView = ({
    display = "page",
    onClose,
    zIndex,
}: InboxViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const { palette } = useTheme();

    const { currTab, handleTabChange, searchType, tabs, title, where } = useTabs<InboxPageTabOption>(tabParams, 0);
    console.log("usetabs data", tabParams, currTab, handleTabChange, searchType, tabs, title, where);

    const {
        allData,
        loading,
        loadMore,
        setAllData,
    } = useFindMany<InboxObject>({
        canSearch: () => true,
        searchType,
        where,
    });

    const [deleteMutation, { error: deleteError }] = useCustomMutation<Success, DeleteOneInput>(deleteOneOrManyDeleteOne);
    const [markAsReadMutation, { error: markError }] = useCustomMutation<Success, FindByIdInput>(notificationMarkAsRead);
    const [markAllAsReadMutation, { error: markAllError }] = useCustomMutation<Success, undefined>(notificationMarkAllAsRead);
    useDisplayApolloError(deleteError ?? markError ?? markAllError);

    const openNotification = useCallback((notification: Notification) => {
        if (notification.link) {
            openLink(setLocation, notification.link);
        }
    }, [setLocation]);

    const openChat = useCallback((chat: Chat) => {
        openObject(chat, setLocation);
    }, [setLocation]);

    const onDelete = useCallback((id: string, objectType: InboxType) => {
        mutationWrapper<Success, DeleteOneInput>({
            mutation: deleteMutation,
            input: { id, objectType: objectType as DeleteType },
            successCondition: (data) => data.success,
            onSuccess: () => {
                setAllData(n => n.filter(n => n.id !== id));
            },
        });
    }, [deleteMutation, setAllData]);

    const onMarkAsRead = useCallback((id: string, objectType: InboxType) => {
        // TODO handle chats
        if (objectType === "Chat") return;
        mutationWrapper<Success, FindByIdInput>({
            mutation: markAsReadMutation,
            input: { id },
            successCondition: (data) => data.success,
            onSuccess: () => {
                setAllData(n => n.map(n => n.id === id ? { ...n, isRead: true } : n));
            },
        });
    }, [markAsReadMutation, setAllData]);

    const onMarkAllAsRead = useCallback(() => {
        // TODO handle chats
        mutationWrapper<Success, any>({
            mutation: markAllAsReadMutation,
            successCondition: (data) => data.success,
            onSuccess: () => {
                setAllData(n => n.map(n => ({ ...n, isRead: true })));
            },
        });
    }, [markAllAsReadMutation, setAllData]);

    const onAction = useCallback((action: keyof (ChatListItemActions | NotificationListItemActions), id: string) => {
        switch (action) {
            case "MarkAsRead":
                onMarkAsRead(id, searchType as InboxType);
                break;
            case "Delete":
                onDelete(id, searchType as InboxType);
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

    const listItems = useMemo(() => listToListItems<InboxType>({
        dummyItems: new Array(5).fill(searchType),
        items: allData,
        keyPrefix: `${searchType}-list-item`,
        loading,
        onAction,
        onClick: (item) => onClick(item as InboxObject),
        zIndex: 200,
    }), [allData, searchType, loading, onAction, onClick]);

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
            <TopBar
                display={display}
                onClose={onClose}
                titleData={{
                    title,
                }}
                below={<PageTabs
                    ariaLabel="inbox-tabs"
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            <Button
                onClick={onMarkAllAsRead}
                disabled={!listItems.length}
                sx={{
                    marginLeft: "auto",
                    marginRight: "auto",
                    marginTop: 2,
                    marginBottom: 2,
                    display: "block",
                }}
            >{t("MarkAllAsRead")}</Button>
            <ListContainer
                emptyText={t("NoResults", { ns: "error" })}
                isEmpty={listItems.length === 0}
            >
                {listItems}
            </ListContainer>
            {/* New Chat button */}
            {currTab.value === InboxPageTabOption.Messages && <SideActionButtons
                display={display}
                zIndex={zIndex + 1}
                sx={{ position: "fixed" }}
            >
                <ColorIconButton aria-label="new-chat" background={palette.secondary.main} onClick={() => { }} >
                    <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </ColorIconButton>
            </SideActionButtons>}
        </>
    );
};
