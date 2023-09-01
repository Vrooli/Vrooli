import { Chat, DeleteOneInput, DeleteType, endpointPostDeleteOne, endpointPutNotification, endpointPutNotificationsMarkAllAsRead, FindByIdInput, Notification, Success } from "@local/shared";
import { IconButton, Tooltip, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { ListActions } from "components/lists/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useDisplayServerError } from "hooks/useDisplayServerError";
import { useFindMany } from "hooks/useFindMany";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useTabs } from "hooks/useTabs";
import { AddIcon, CompleteIcon } from "icons";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ListObject } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { InboxPageTabOption, inboxTabParams } from "utils/search/objectToSearch";
import { ChatUpsert } from "views/objects/chat/ChatUpsert/ChatUpsert";
import { InboxViewProps } from "../types";

type InboxType = "Chat" | "Notification";
type InboxObject = Chat | Notification;

export const InboxView = ({
    isOpen,
    onClose,
}: InboxViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const { palette } = useTheme();
    const display = toDisplay(isOpen);

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs<InboxPageTabOption>({ tabParams: inboxTabParams, display });

    const {
        allData,
        loading,
        loadMore,
        setAllData,
    } = useFindMany<InboxObject>({
        searchType,
        where: where(),
    });
    console.log("alldata", allData);

    const [deleteMutation, { errors: deleteErrors }] = useLazyFetch<DeleteOneInput, Success>(endpointPostDeleteOne);
    const [markAsReadMutation, { errors: markErrors }] = useLazyFetch<FindByIdInput, Success>(endpointPutNotification);
    const [markAllAsReadMutation, { errors: markAllErrors }] = useLazyFetch<undefined, Success>(endpointPutNotificationsMarkAllAsRead);
    useDisplayServerError(deleteErrors ?? markErrors ?? markAllErrors);

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
        if (currTab.tabType === InboxPageTabOption.Notification) {
            return [onMarkAllAsRead, CompleteIcon, "MarkAllAsRead"] as const;
        }
        return [openCreateChat, AddIcon, "CreateChat"] as const;
    }, [currTab.tabType, onMarkAllAsRead, openCreateChat]);

    const onAction = useCallback((action: keyof ListActions, data: unknown) => {
        switch (action) {
            case "Delete":
                onDelete(data as InboxType, searchType as InboxType);
                break;
            case "MarkAsRead":
                onMarkAsRead(data as string, searchType as InboxType);
                break;
        }
    }, [onDelete, onMarkAsRead, searchType]);

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
            />
            {/* Main content */}
            <TopBar
                display={display}
                hideTitleOnDesktop={true}
                onClose={onClose}
                title={currTab.label}
                below={<PageTabs
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
            >
                <ObjectList
                    dummyItems={new Array(5).fill(searchType)}
                    items={allData as ListObject[]}
                    keyPrefix={`${searchType}-list-item`}
                    loading={loading}
                    onAction={onAction}
                />
            </ListContainer>
            {/* New Chat button */}
            <SideActionsButtons
                display={display}
                sx={{ position: "fixed" }}
            >
                <Tooltip title={t(actionTooltip)}>
                    <IconButton aria-label={t("CreateChat")} onClick={onActionButtonPress} sx={{ background: palette.secondary.main }}>
                        <ActionButtonIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </IconButton>
                </Tooltip>
            </SideActionsButtons>
        </>
    );
};
