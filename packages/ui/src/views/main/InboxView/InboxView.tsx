import { Chat, endpointPutNotificationsMarkAllAsRead, Notification, Success } from "@local/shared";
import { IconButton, Tooltip, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { ObjectListActions } from "components/lists/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useFindMany } from "hooks/useFindMany";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useTabs } from "hooks/useTabs";
import { AddIcon, CompleteIcon } from "icons";
import { useCallback, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { pagePaddingBottom } from "styles";
import { ArgsType } from "types";
import { ListObject } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { getObjectUrlBase } from "utils/navigation/openObject";
import { InboxPageTabOption, inboxTabParams } from "utils/search/objectToSearch";
import { InboxViewProps } from "../types";

type InboxObject = Chat | Notification;

export const InboxView = ({
    isOpen,
    onClose,
}: InboxViewProps) => {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const display = toDisplay(isOpen);

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs<InboxPageTabOption>({ id: "inbox-tabs", tabParams: inboxTabParams, display });

    const {
        allData,
        loading,
        removeItem,
        loadMore,
        setAllData,
        updateItem,
    } = useFindMany<InboxObject>({
        searchType,
        where: where(),
    });
    console.log("alldata", allData);


    const [markAllAsReadMutation] = useLazyFetch<undefined, Success>(endpointPutNotificationsMarkAllAsRead);

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
        if (currTab.tabType === InboxPageTabOption.Notification) {
            return [onMarkAllAsRead, CompleteIcon, "MarkAllAsRead"] as const;
        }
        return [openCreateChat, AddIcon, "CreateChat"] as const;
    }, [currTab.tabType, onMarkAllAsRead, openCreateChat]);

    const onAction = useCallback((action: keyof ObjectListActions<InboxObject>, ...data: unknown[]) => {
        console.log("inboxview onaction", action, data);
        switch (action) {
            case "Deleted":
                removeItem(...(data as ArgsType<ObjectListActions<InboxObject>["Deleted"]>));
                break;
            case "Updated":
                updateItem(...(data as ArgsType<ObjectListActions<InboxObject>["Updated"]>));
                break;
        }
    }, [removeItem, updateItem]);

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
                sx={{ marginBottom: pagePaddingBottom }}
            >
                <ObjectList
                    dummyItems={new Array(5).fill(searchType)}
                    items={allData as ListObject[]}
                    keyPrefix={`${searchType}-list-item`}
                    loading={loading}
                    onAction={onAction}
                />
            </ListContainer>
            <SideActionsButtons display={display}>
                <Tooltip title={t(actionTooltip)}>
                    <IconButton aria-label={t("CreateChat")} onClick={onActionButtonPress} sx={{ background: palette.secondary.main }}>
                        <ActionButtonIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </IconButton>
                </Tooltip>
            </SideActionsButtons>
        </>
    );
};
