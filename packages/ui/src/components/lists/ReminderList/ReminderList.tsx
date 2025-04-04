import { DragDropContext, Draggable, DropResult, Droppable } from "@hello-pangea/dnd";
import { Count, DUMMY_ID, DeleteManyInput, DeleteType, LINKS, ListObject, Reminder, ResourceUsedFor, endpointsActions, updateArray } from "@local/shared";
import { Box, IconButton, Tooltip, Typography, styled } from "@mui/material";
import { SyntheticEvent, forwardRef, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { SessionContext } from "../../../contexts/session.js";
import { useBulkObjectActions } from "../../../hooks/objectActions.js";
import { useLazyFetch } from "../../../hooks/useLazyFetch.js";
import { useSelectableList } from "../../../hooks/useSelectableList.js";
import { IconCommon } from "../../../icons/Icons.js";
import { openLink } from "../../../route/openLink.js";
import { useLocation } from "../../../route/router.js";
import { useFocusModes } from "../../../stores/focusModeStore.js";
import { multiLineEllipsis, noSelect } from "../../../styles.js";
import { ArgsType } from "../../../types.js";
import { DUMMY_LIST_LENGTH_SHORT, ELEMENT_IDS } from "../../../utils/consts.js";
import { getResourceIcon } from "../../../utils/display/getResourceIcon.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { firstString } from "../../../utils/display/stringTools.js";
import { getUserLanguages } from "../../../utils/display/translationTools.js";
import { PubSub } from "../../../utils/pubsub.js";
import { ReminderCrud, reminderInitialValues } from "../../../views/objects/reminder/ReminderCrud.js";
import { TextLoading } from "../TextLoading/TextLoading.js";
import { ObjectListActions, ReminderCardProps, ReminderListHorizontalProps, ReminderListProps } from "../types.js";

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
const CardBox = styled(Box)(({ theme }) => ({
    ...noSelect,
    alignItems: "center",
    background: theme.palette.background.paper,
    borderRadius: theme.spacing(2),
    boxShadow: theme.shadows[2],
    color: theme.palette.background.textSecondary,
    display: "flex",
    flexDirection: "row",
    gap: theme.spacing(1),
    height: "60px",
    cursor: "pointer",
    maxWidth: "250px",
    minWidth: "120px",
    padding: theme.spacing(1),
    position: "relative",
    textDecoration: "none",
    width: "fit-content",
    "&:hover": {
        filter: "brightness(120%)",
        transition: "filter 0.2s",
    },
    "&.past": {
        opacity: 0.6,
        filter: "grayscale(30%)",
        "&:hover": {
            opacity: 0.8,
            filter: "grayscale(0%)",
            transition: "all 0.2s",
        },
    },
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

const ReminderCard = forwardRef<unknown, ReminderCardProps>(({
    data,
    dragProps,
    dragHandleProps,
    isEditing,
    onEdit,
    onDelete,
}, ref) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const { title, subtitle } = useMemo(() => {
        const { title, subtitle } = getDisplay(data, getUserLanguages(session));
        return {
            title: title ? title : t((data.usedFor ?? "Context") as TranslationKeyCommon),
            subtitle,
        };
    }, [data, session, t]);

    const Icon = useMemo(() => {
        return getResourceIcon({
            fill: "background.textSecondary",
            link: data.link,
            usedFor: data.usedFor ?? ResourceUsedFor.Related,
        });
    }, [data.link, data.usedFor]);

    const href = useMemo(() => getResourceUrl(data.link), [data]);
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
            console.error("[ReminderCard] Unhandled click event", event.target);
        }
    }, [data, href, isEditing, onDelete, onEdit, setLocation]);

    return (
        <Tooltip placement="top" title={`${subtitle ? subtitle + " - " : ""}${data.link}`}>
            <CardBox
                ref={ref}
                {...dragProps}
                {...dragHandleProps}
                component="a"
                href={href}
                onClick={handleClick}
            >
                {Icon}
                <Typography
                    variant="caption"
                    sx={titleStyle}
                >
                    {firstString(title, data.link)}
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
ReminderCard.displayName = "ReminderCard";

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

function ReminderListHorizontal({
    canUpdate = true,
    handleUpdate,
    id,
    isEditing,
    list,
    loading,
    onDelete,
    openAddDialog,
    openUpdateDialog,
}: ReminderListHorizontalProps) {
    const { t } = useTranslation();

    const onDragEnd = useCallback((result: DropResult) => {
        const { source, destination } = result;
        if (!isEditing || !destination || source.index === destination.index) return;
        // Handle the reordering of the reminders in the list
        if (handleUpdate && list) {
            handleUpdate({
                ...list,
                reminders: updateArray(list.reminders, source.index, list.reminders[destination.index]),
            });
        }
    }, [isEditing, handleUpdate, list]);

    const isEmpty = !list?.reminders?.length;

    // If empty, not loading, and can't add an item, show "No reminders"
    if (isEmpty && !loading && !canUpdate) {
        return <Typography variant="body1" color="text.secondary">No reminders</Typography>;
    }
    // If empty but loading, show loading cards
    if (isEmpty && loading) {
        return (
            <>
                <CardsBox
                    id={id ?? ELEMENT_IDS.ReminderCards}
                >
                    {Array.from(Array(DUMMY_LIST_LENGTH_SHORT).keys()).map((i) => (
                        <LoadingCard key={`reminder-card-${i}`} />
                    ))}
                </CardsBox>
            </>
        );
    }
    // Otherwise, show drag/drop list with item cards, loading cards (if relavant), and add card (if relevant)
    return (
        <>
            <DragDropContext onDragEnd={onDragEnd}>
                <Droppable droppableId="reminder-list" direction="horizontal">
                    {(providedDrop) => (
                        <CardsBox
                            ref={providedDrop.innerRef}
                            id={id}
                            {...providedDrop.droppableProps}
                        >
                            {/* Reminders */}
                            {list?.reminders?.map((c: Reminder, index) => (
                                <Draggable
                                    key={`reminder-card-${index}`}
                                    draggableId={`reminder-card-${index}`}
                                    index={index}
                                    isDragDisabled={!isEditing}
                                >
                                    {(providedDrag) => (
                                        <ReminderCard
                                            ref={providedDrag.innerRef}
                                            dragProps={providedDrag.draggableProps}
                                            dragHandleProps={providedDrag.dragHandleProps}
                                            key={`reminder-card-${index}`}
                                            isEditing={isEditing}
                                            data={{ ...c, list }}
                                            onEdit={openUpdateDialog}
                                            onDelete={onDelete}
                                        />
                                    )}
                                </Draggable>
                            ))}
                            {/* Dummy cards when loading */}
                            {
                                loading && !Array.isArray(list?.reminders) && Array.from(Array(DUMMY_LIST_LENGTH_SHORT).keys()).map((i) => (
                                    <LoadingCard key={`reminder-card-${i}`} />
                                ))
                            }
                            {/* Add button */}
                            {canUpdate ? <Tooltip placement="top" title={t("AddReminder")}>
                                <CardBox onClick={openAddDialog}>
                                    <IconCommon name="Add" size={ICON_SIZE} />
                                    <Typography
                                        variant="caption"
                                        component="span"
                                        sx={titleStyle}
                                    >
                                        {t("AddReminder")}
                                    </Typography>
                                </CardBox>
                            </Tooltip> : null}
                            {providedDrop.placeholder}
                        </CardsBox>
                    )}
                </Droppable>
            </DragDropContext>
        </>
    );
}

export function ReminderList(props: ReminderListProps) {
    const { canUpdate, handleUpdate, list, mutate, title } = props;
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const session = useContext(SessionContext);
    const focusModeInfo = useFocusModes(session);

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
    const openUpdateDialog = useCallback((reminder: Reminder) => {
        const index = list?.reminders?.findIndex(r => r.id === reminder.id) ?? -1;
        setEditingIndex(index);
        setIsAddDialogOpen(true);
    }, [list?.reminders]);
    const onCompleted = useCallback((reminder: Reminder) => {
        closeAddDialog();
        if (!list || !handleUpdate) return;
        const index = reminder.index;
        if (index && index >= 0) {
            handleUpdate({
                ...list,
                reminders: updateArray(list.reminders, index, reminder),
            });
        }
        else {
            handleUpdate({
                ...list,
                reminders: [...(list?.reminders ?? []), reminder],
            });
        }
    }, [closeAddDialog, handleUpdate, list]);
    const onDeleted = useCallback((reminder: Reminder) => {
        closeAddDialog();
        if (!list || !handleUpdate) return;
        handleUpdate({
            ...list,
            reminders: list.reminders.filter(r => r.id !== reminder.id),
        });
    }, [closeAddDialog, handleUpdate, list]);

    const [deleteMutation] = useLazyFetch<DeleteManyInput, Count>(endpointsActions.deleteMany);
    const onDelete = useCallback((item: Reminder) => {
        if (!list) return;
        if (mutate && item.id) {
            fetchLazyWrapper<DeleteManyInput, Count>({
                fetch: deleteMutation,
                inputs: { objects: [{ id: item.id, objectType: DeleteType.Reminder }] },
                onSuccess: () => {
                    handleUpdate?.({
                        ...list,
                        reminders: list.reminders.filter(r => r.id !== item.id),
                    });
                },
            });
        }
        else if (handleUpdate) {
            handleUpdate({
                ...list,
                reminders: list.reminders.filter(r => r.id !== item.id),
            });
        }
    }, [deleteMutation, handleUpdate, list, mutate]);

    const upsertDialog = useMemo(() => {
        if (!isAddDialogOpen) return null;

        const overrideObject = editingIndex >= 0 && list?.reminders
            ? { ...list.reminders[editingIndex as number], index: editingIndex }
            : reminderInitialValues(focusModeInfo, {
                index: 0,
                list: {
                    __connect: true,
                    ...(list?.id && list.id !== DUMMY_ID ? list : { listFor: parent }),
                    id: list?.id ?? DUMMY_ID,
                    __typename: "ReminderList",
                },
            }) as Reminder;

        return <ReminderCrud
            display="dialog"
            isCreate={editingIndex < 0}
            isOpen={isAddDialogOpen}
            onCancel={closeAddDialog}
            onClose={closeAddDialog}
            onCompleted={onCompleted}
            onDeleted={onDeleted}
            overrideObject={overrideObject}
        />;
    }, [closeAddDialog, editingIndex, focusModeInfo, isAddDialogOpen, list, onCompleted, onDeleted]);

    const {
        isSelecting,
        handleToggleSelecting,
        handleToggleSelect,
        selectedData,
        setIsSelecting,
        setSelectedData,
    } = useSelectableList<Reminder>(list?.reminders ?? []);
    const { onBulkActionStart, BulkDeleteDialogComponent } = useBulkObjectActions<Reminder>({
        allData: list?.reminders ?? [],
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
    const onAction = useCallback((action: keyof ObjectListActions<Reminder>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted": {
                if (!list) return;
                const [id] = data as ArgsType<ObjectListActions<Reminder>["Deleted"]>;
                const item = list?.reminders?.find(r => r.id === id);
                if (!item) {
                    PubSub.get().publish("snack", { message: "Item not found", severity: "Warning" });
                    return;
                }
                onDeleted(item);
                break;
            }
            case "Updated": {
                if (!list) return;
                const [updatedItem] = data as ArgsType<ObjectListActions<Reminder>["Updated"]>;
                handleUpdate?.({
                    ...list,
                    reminders: list.reminders.map(r => r.id === updatedItem.id ? updatedItem : r),
                });
                break;
            }
        }
    }, [handleUpdate, list, onDeleted]);
    const onClick = useCallback((data: ListObject) => {
        //TODO
    }, []);

    const childProps: ReminderListHorizontalProps = {
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
        <Box>
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
            <ReminderListHorizontal {...childProps} />
        </Box>
    );
}
