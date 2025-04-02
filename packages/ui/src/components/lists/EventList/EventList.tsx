import { CalendarEvent, Count, DeleteManyInput, LINKS, ListObject, Resource, ResourceUsedFor, endpointsActions, getObjectUrl } from "@local/shared";
import { Box, IconButton, Tooltip, Typography, styled } from "@mui/material";
import { SyntheticEvent, forwardRef, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../../contexts/session.js";
import { useBulkObjectActions } from "../../../hooks/objectActions.js";
import { useLazyFetch } from "../../../hooks/useLazyFetch.js";
import { useSelectableList } from "../../../hooks/useSelectableList.js";
import { Icon, IconCommon, IconInfo } from "../../../icons/Icons.js";
import { openLink } from "../../../route/openLink.js";
import { useLocation } from "../../../route/router.js";
import { CardBox, multiLineEllipsis } from "../../../styles.js";
import { ArgsType } from "../../../types.js";
import { DUMMY_LIST_LENGTH_SHORT, ELEMENT_IDS } from "../../../utils/consts.js";
import { getResourceIcon } from "../../../utils/display/getResourceIcon.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { firstString } from "../../../utils/display/stringTools.js";
import { getUserLanguages } from "../../../utils/display/translationTools.js";
import { PubSub } from "../../../utils/pubsub.js";
import { TextLoading } from "../TextLoading/TextLoading.js";
import { EventCardProps, EventListHorizontalProps, EventListProps, ObjectListActions } from "../types.js";

const CONTENT_SPACING = 1;
const ICON_SIZE = 20;

const CardsBox = styled(Box)(({ theme }) => ({
    alignItems: "flex-start",
    display: "flex",
    flexWrap: "wrap",
    gap: theme.spacing(1),
    paddingBottom: 1,
    justifyContent: "flex-start",
    width: "100%",
}));
const PlaceholderIcon = styled(Box)(({ theme }) => ({
    width: ICON_SIZE,
    height: ICON_SIZE,
    borderRadius: "16px",
    backgroundColor: theme.palette.text.primary,
    opacity: 0.2,
}));

const titleStyle = {
    ...multiLineEllipsis(1),
    flexGrow: 1,
} as const;

const loadingCardStyle = {
    display: "flex",
    alignItems: "center",
    gap: CONTENT_SPACING,
    width: "100%",
} as const;

const loadingTextStyle = {
    width: "60%",
    flexGrow: 1,
    opacity: 0.3,
} as const;

const EventCard = forwardRef<unknown, EventCardProps>(({
    data,
    isEditing,
    onEdit,
    onDelete,
}, ref) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();

    const { title, subtitle, DisplayIcon } = useMemo(() => {
        const scheduleOn =
            data.schedule.focusModes?.[0] ??
            data.schedule.meetings?.[0] ??
            data.schedule.runProjects?.[0] ??
            data.schedule.runRoutines?.[0];

        const scheduleDisplay = getDisplay(data.schedule, getUserLanguages(session));
        const scheduleOnDisplay = getDisplay(scheduleOn, getUserLanguages(session));
        const title = data.title || scheduleDisplay.title || scheduleOnDisplay.title;
        const subtitle = scheduleDisplay.subtitle || scheduleOnDisplay.subtitle;

        const fallbackIconInfo: IconInfo = { name: "Schedule", type: "Common" };
        let DisplayIcon: JSX.Element | null = null;
        if (!data.schedule) {
            DisplayIcon = <Icon info={fallbackIconInfo} />;
        }
        else if (!scheduleOn) {
            DisplayIcon = <Icon info={fallbackIconInfo} />;
        }
        else {
            DisplayIcon = getResourceIcon({
                fill: "background.textSecondary",
                link: `${window.location.origin}${getObjectUrl(scheduleOn)}`,
                usedFor: ResourceUsedFor.Context,
            });
        }
        return {
            title,
            subtitle,
            DisplayIcon,
        };
    }, [data, session]);

    const href = useMemo(() => getObjectUrl(data), [data]);
    const handleClick = useCallback((event: SyntheticEvent) => {
        event.preventDefault();
        const target = event.target as HTMLElement | SVGElement | null;
        if (!target) return;
        const classList = target.classList;
        const hasDeleteClass = Array.from(classList).some(c => c.startsWith("delete-"));
        if (hasDeleteClass) {
            onDelete?.(data);
        }
        else if (isEditing) {
            onEdit?.(data);
        } else if (href) {
            openLink(setLocation, href);
        } else {
            console.error("[EventCard] Unhandled click event", event.target);
        }
    }, [data, href, isEditing, onDelete, onEdit, setLocation]);

    return (
        <Tooltip placement="top" title={subtitle}>
            <CardBox
                ref={ref}
                component="a"
                href={href}
                onClick={handleClick}
            >
                {DisplayIcon}
                <Typography
                    variant="caption"
                    sx={titleStyle}
                >
                    {firstString(title, subtitle)}
                </Typography>
                {isEditing && <IconButton className="delete-icon-button">
                    <IconCommon
                        className="delete-icon"
                        fill="background.textSecondary"
                        name="Delete"
                    />
                </IconButton>}
            </CardBox>
        </Tooltip>
    );
});
EventCard.displayName = "EventCard";

function LoadingCard() {
    return (
        <CardBox>
            <Box sx={loadingCardStyle}>
                <PlaceholderIcon />
                <TextLoading size="caption" lines={1} sx={loadingTextStyle} />
            </Box>
        </CardBox>
    );
}

function EventListHorizontal({
    canUpdate = true,
    handleUpdate,
    id,
    isEditing,
    list,
    loading,
    onDelete,
    openAddDialog,
    openUpdateDialog,
}: EventListHorizontalProps) {
    const { t } = useTranslation();

    const isEmpty = !list?.length;

    // If empty, not loading, and can't add an item, show "No upcoming events"
    if (isEmpty && !loading && !canUpdate) {
        return <Typography variant="body1" color="text.secondary">No upcoming events</Typography>;
    }
    // If empty but loading, show loading cards
    if (isEmpty && loading) {
        return (
            <>
                <CardsBox
                    id={id ?? ELEMENT_IDS.EventCards}
                >
                    {Array.from(Array(DUMMY_LIST_LENGTH_SHORT).keys()).map((i) => (
                        <LoadingCard key={`event-card-${i}`} />
                    ))}
                </CardsBox>
            </>
        );
    }
    // Otherwise, show list with item cards, loading cards (if relavant), and add card (if relevant)
    return (
        <CardsBox id={id}>
            {/* Events */}
            {list?.map((c: CalendarEvent, index) => (
                <EventCard
                    key={`event-card-${index}`}
                    isEditing={isEditing}
                    data={{ ...c, list }}
                    onEdit={openUpdateDialog}
                    onDelete={onDelete}
                />
            ))}
            {/* Dummy cards when loading */}
            {
                loading && !Array.isArray(list) && Array.from(Array(DUMMY_LIST_LENGTH_SHORT).keys()).map((i) => (
                    <LoadingCard key={`event-card-${i}`} />
                ))
            }
            {/* Add button */}
            {canUpdate ? <Tooltip placement="top" title={t("AddEvent")}>
                <CardBox onClick={openAddDialog}>
                    <IconCommon name="Add" size={ICON_SIZE} />
                    <Typography
                        variant="caption"
                        component="span"
                        sx={titleStyle}
                    >
                        {t("AddEvent")}
                    </Typography>
                </CardBox>
            </Tooltip> : null}
        </CardsBox>
    );
}

export function EventList(props: EventListProps) {
    const { canUpdate, handleUpdate, list, mutate, title } = props;
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const [isEditing, setIsEditing] = useState(false);
    useEffect(() => {
        if (!canUpdate) setIsEditing(false);
    }, [canUpdate]);

    function openSchedule() {
        setLocation(LINKS.Calendar);
    }
    function toggleEditing() {
        setIsEditing(e => !e);
    }

    // Upsert dialog
    const [editingIndex, setEditingIndex] = useState<number>(-1);
    const [isAddDialogOpen, setIsAddDialogOpen] = useState(false);
    const openAddDialog = useCallback(() => { setIsAddDialogOpen(true); }, []);
    const closeAddDialog = useCallback(() => { setIsAddDialogOpen(false); setEditingIndex(-1); }, []);
    const openUpdateDialog = useCallback((calendarEvent: CalendarEvent) => {
        const index = list?.findIndex(c => c.id === calendarEvent.id) ?? -1;
        setEditingIndex(index);
        setIsAddDialogOpen(true);
    }, [list]);
    const onCompleted = useCallback((calendarEvent: CalendarEvent) => {
        closeAddDialog();
        // if (!list || !handleUpdate) return;
        // const index = resource.index;
        // if (index && index >= 0) {
        //     handleUpdate({
        //         ...list,
        //         resources: updateArray(list.resources, index, resource),
        //     });
        // }
        // else {
        //     handleUpdate({
        //         ...list,
        //         resources: [...(list?.resources ?? []), resource],
        //     });
        // }
    }, [closeAddDialog, handleUpdate, list]);
    const onDeleted = useCallback((calendarEvent: CalendarEvent) => {
        closeAddDialog();
        if (!list || !handleUpdate) return;
        handleUpdate([...list.filter(c => c.id !== calendarEvent.id)]);
    }, [closeAddDialog, handleUpdate, list]);

    const [deleteMutation] = useLazyFetch<DeleteManyInput, Count>(endpointsActions.deleteMany);
    const onDelete = useCallback((item: Resource) => {
        return;
        // if (!list) return;
        // if (mutate && item.id) {
        //     fetchLazyWrapper<DeleteManyInput, Count>({
        //         fetch: deleteMutation,
        //         inputs: { objects: [{ id: item.id, objectType: DeleteType.Resource }] },
        //         onSuccess: () => {
        //             handleUpdate?.({
        //                 ...list,
        //                 resources: list.resources.filter(r => r.id !== item.id),
        //             });
        //         },
        //     });
        // }
        // else if (handleUpdate) {
        //     handleUpdate({
        //         ...list,
        //         resources: list.resources.filter(r => r.id !== item.id),
        //     });
        // }
    }, [deleteMutation, handleUpdate, list, mutate]);

    const upsertDialog = null;
    // const upsertDialog = useMemo(() => (
    //     isAddDialogOpen ? <ResourceUpsert
    //         display="dialog"
    //         isCreate={editingIndex < 0}
    //         isOpen={isAddDialogOpen}
    //         isMutate={mutate ?? false}
    //         onCancel={closeAddDialog}
    //         onClose={closeAddDialog}
    //         onCompleted={onCompleted}
    //         onDeleted={onDeleted}
    //         overrideObject={(editingIndex >= 0 && list?.resources ?
    //             { ...list.resources[editingIndex as number], index: editingIndex } :
    //             resourceInitialValues(undefined, {
    //                 index: 0,
    //                 list: {
    //                     __connect: true,
    //                     ...(list?.id && list.id !== DUMMY_ID ? list : { listFor: parent }),
    //                     id: list?.id ?? DUMMY_ID,
    //                     __typename: "EventList",
    //                 },
    //             }) as Resource)}
    //     /> : null
    // ), [closeAddDialog, editingIndex, isAddDialogOpen, list, mutate, onCompleted, onDeleted, parent]);

    const {
        isSelecting,
        handleToggleSelecting,
        handleToggleSelect,
        selectedData,
        setIsSelecting,
        setSelectedData,
    } = useSelectableList<Resource>(list ?? []);
    const { onBulkActionStart, BulkDeleteDialogComponent } = useBulkObjectActions<Resource>({
        allData: list ?? [],
        selectedData,
        setAllData: (data) => {
            //TODO
        },
        setSelectedData: (data) => {
            setSelectedData(data);
            setIsSelecting(false);
        },
        setLocation,
    });
    const onAction = useCallback((action: keyof ObjectListActions<Resource>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted": {
                if (!list) return;
                const [id] = data as ArgsType<ObjectListActions<Resource>["Deleted"]>;
                const item = list?.find(r => r.id === id);
                if (!item) {
                    PubSub.get().publish("snack", { message: "Item not found", severity: "Warning" });
                    return;
                }
                onDeleted(item);
                break;
            }
            case "Updated": {
                if (!list) return;
                const [updatedItem] = data as ArgsType<ObjectListActions<Resource>["Updated"]>;
                handleUpdate?.({
                    ...list,
                    resources: list.map(r => r.id === updatedItem.id ? updatedItem : r),
                });
                break;
            }
        }
    }, [handleUpdate, list, onDeleted]);
    const onClick = useCallback((data: ListObject) => {
        //TODO
    }, []);

    const childProps: EventListHorizontalProps = {
        ...props,
        handleToggleSelect,
        isEditing,
        isSelecting,
        onAction,
        onClick,
        onDelete,
        openAddDialog,
        openUpdateDialog,
        selectedData,
    };

    return (
        <>
            {upsertDialog}
            {BulkDeleteDialogComponent}
            {title && <Box display="flex" flexDirection="row" alignItems="center">
                <Typography component="h2" variant="h6" textAlign="left">{title}</Typography>
                <Tooltip title={t("Open")}>
                    <IconButton onClick={openSchedule}>
                        <IconCommon name="OpenInNew" fill="secondary.main" size={24} />
                    </IconButton>
                </Tooltip>
                <Tooltip title={t("Edit")}>
                    <IconButton onClick={toggleEditing}>
                        <IconCommon name={isEditing ? "Close" : "Edit"} fill="secondary.main" size={24} />
                    </IconButton>
                </Tooltip>
            </Box>}
            <EventListHorizontal {...childProps} />
        </>
    );
}
