/**
 * Displays all search options for a team
 */
import { Bookmark, BookmarkCreateInput, BookmarkFor, BookmarkList, BookmarkListCreateInput, BookmarkSearchInput, BookmarkSearchResult, Count, DeleteManyInput, DeleteType, endpointsActions, endpointsBookmark, endpointsBookmarkList, lowercaseFirstLetter, shapeBookmark, shapeBookmarkList, uuid } from "@local/shared";
import { Box, Button, Checkbox, Divider, IconButton, List, ListItem, ListItemButton, ListItemIcon, ListItemText, TextField, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api/fetchWrapper.js";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons.js";
import { useBookmarkListsStore } from "hooks/objectActions.js";
import { useLazyFetch } from "hooks/useLazyFetch.js";
import { AddIcon, ArrowLeftIcon } from "icons/common.js";
import { useCallback, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { ELEMENT_IDS } from "utils/consts.js";
import { SessionContext } from "../../../contexts.js";
import { LargeDialog } from "../LargeDialog/LargeDialog.js";
import { SelectBookmarkListDialogProps } from "../types.js";

const createListIconButtonStyle = {
    marginLeft: "auto",
} as const;
const bookmarkListStyle = {
    // Make sure the list is display above both the BottomNav and any 
    // snack messages
    paddingBottom: "calc(128px + env(safe-area-inset-bottom))",
} as const;

export function SelectBookmarkListDialog({
    objectId,
    objectType,
    onClose,
    isCreate,
    isOpen,
}: SelectBookmarkListDialogProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    const bookmarkLists = useBookmarkListsStore(state => state.bookmarkLists);

    const [lists, setLists] = useState<BookmarkList[]>(bookmarkLists);
    const [selectedLists, setSelectedLists] = useState<BookmarkList[]>([]);

    useEffect(() => {
        setLists(bookmarkLists);
    }, [bookmarkLists, session]);

    // Fetch all bookmarks for object
    const [refetch, { data, loading: isFindLoading }] = useLazyFetch<BookmarkSearchInput, BookmarkSearchResult>({
        ...endpointsBookmark.findMany,
        inputs: { [`${lowercaseFirstLetter(objectType)}Id`]: objectId },
    });
    useEffect(() => {
        if (!isCreate && isOpen) {
            refetch();
        } else {
            setSelectedLists([]);
        }
    }, [refetch, isCreate, isOpen, objectId]);
    useEffect(() => {
        if (data) {
            setSelectedLists(data.edges.map(e => e.node.list));
        }
    }, [data]);

    const [create, { loading: isCreateLoading }] = useLazyFetch<BookmarkCreateInput, Bookmark>(endpointsBookmark.createOne);
    const [deleteMutation, { loading: isDeleteLoading }] = useLazyFetch<DeleteManyInput, Count>(endpointsActions.deleteMany);
    const handleSubmit = useCallback(async () => {
        // Iterate over selected lists
        for (const list of selectedLists) {
            // If the list was not already selected, add the bookmark
            if (isCreate || !data?.edges.some(e => e.node.id === list.id)) {
                await create(shapeBookmark.create({
                    __typename: "Bookmark",
                    id: uuid(),
                    to: {
                        __typename: objectType as BookmarkFor,
                        id: objectId!,
                    },
                    list: {
                        __typename: "BookmarkList",
                        id: list.id,
                    },
                }));
            }
        }
        // Iterate over non-selected lists
        const deletedBookmarks = data?.edges.filter(e => !selectedLists.some(sl => sl.id === e.node.list.id));
        if (deletedBookmarks) {
            await deleteMutation({
                objects: deletedBookmarks.map(e => ({
                    id: e.node.id,
                    objectType: DeleteType.Bookmark,
                })),
            });
        }
        onClose(selectedLists.length > 0);
    }, [selectedLists, create, data?.edges, deleteMutation, isCreate, objectId, objectType, onClose]);

    const onCancel = useCallback(() => {
        setSelectedLists([]);
        onClose(selectedLists.length > 0);
    }, [onClose, selectedLists.length, setSelectedLists]);

    const [isCreateOpen, setIsCreateOpen] = useState(false);
    const openCreate = useCallback(() => { setIsCreateOpen(true); }, []);
    const closeCreate = useCallback(() => { setIsCreateOpen(false); }, []);
    const onCreated = useCallback((bookmarkList: BookmarkList) => {
        setLists([...lists, bookmarkList]);
        setSelectedLists([bookmarkList]);
    }, [lists]);

    const handleClose = useCallback(() => {
        const isObjectInABookmarkList = selectedLists.length > 0;
        onClose(isObjectInABookmarkList);
    }, [onClose, selectedLists]);

    const listItems = useMemo(() => lists.sort((a, b) => a.label.localeCompare(b.label)).map(list => {
        function handleClick() {
            if (selectedLists.some(l => l.id === list.id)) {
                setSelectedLists(selectedLists.filter(l => l.id !== list.id));
            } else {
                setSelectedLists([...selectedLists, list]);
            }
        }

        const isSelected = selectedLists.some(l => l.id === list.id);
        const textId = `checkbox-list-label-${list.id}`;
        const checkboxInputProps = { "aria-labelledby": textId } as const;

        return (
            <ListItem key={list.id} disablePadding>
                <ListItemButton onClick={handleClick}>
                    <ListItemIcon>
                        <Checkbox
                            edge="start"
                            checked={isSelected}
                            tabIndex={-1}
                            disableRipple
                            inputProps={checkboxInputProps}
                        />
                    </ListItemIcon>
                    <ListItemText
                        id={textId}
                        primary={`${list.label} (${list.bookmarksCount})`}
                    />
                </ListItemButton>
            </ListItem>
        );
    }), [lists, selectedLists]);

    return (
        <>
            <CreateBookmarkListDialog
                isOpen={isCreateOpen && isOpen}
                onClose={closeCreate}
                onCreated={onCreated}
            />
            <LargeDialog
                id="select-bookmark-list-dialog"
                isOpen={isOpen}
                onClose={handleClose}
                titleId={ELEMENT_IDS.SelectBookmarkListDialog}
            >
                <Box display="flex" flexDirection="row" alignItems="center" p={1}>
                    <Typography variant="h6" ml="auto">{t("AddToList")}</Typography>
                    <IconButton
                        onClick={openCreate}
                        sx={createListIconButtonStyle}
                    >
                        <AddIcon fill={palette.secondary.main} />
                    </IconButton>
                </Box>
                <Divider />
                <List sx={bookmarkListStyle}>
                    {listItems}
                </List>
                <BottomActionsButtons
                    display={"dialog"}
                    isCreate={false}
                    loading={isFindLoading || isCreateLoading || isDeleteLoading}
                    onCancel={onCancel}
                    onSubmit={handleSubmit}
                />
            </LargeDialog>
        </>
    );
}

interface CreateBookmarkListDialogProps {
    isOpen: boolean;
    onClose: () => unknown;
    onCreated: (bookmarkList: BookmarkList) => unknown;
}

function CreateBookmarkListDialog({
    isOpen,
    onClose,
    onCreated,
}: CreateBookmarkListDialogProps) {
    const { t } = useTranslation();
    const [name, setName] = useState("");
    const nameInputRef = useRef<HTMLInputElement>(null);

    const { palette } = useTheme();

    // useLazyFetch for creating a BookmarkList
    const [createList, { loading: isCreating }] = useLazyFetch<BookmarkListCreateInput, BookmarkList>(endpointsBookmarkList.createOne);

    useLayoutEffect(function autoFocusNameInputEffect() {
        if (isOpen && nameInputRef.current) {
            // Focus the input using ref for accessibility
            nameInputRef.current.focus();
        }
    }, [isOpen]);

    const handleCreate = useCallback(async () => {
        if (name.trim() === "") {
            return;
        }
        fetchLazyWrapper<BookmarkListCreateInput, BookmarkList>({
            fetch: createList,
            inputs: shapeBookmarkList.create({
                __typename: "BookmarkList",
                id: uuid(),
                label: name.trim(),
            }),
            onSuccess: (data) => {
                onCreated(data);
                setName("");
                onClose();
            },
        });
    }, [name, createList, onCreated, onClose]);

    const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
        if (e.key === "Enter") {
            handleCreate();
        }
    }, [handleCreate]);
    const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setName(e.target.value);
    }, []);

    const handleBack = useCallback(() => {
        onClose();
    }, [onClose]);

    return (
        <LargeDialog
            id="create-bookmark-list-dialog"
            isOpen={isOpen}
            onClose={onClose}
            titleId="create-bookmark-list"
        >
            <Box display="flex" flexDirection="row" alignItems="center" p={1}>
                <IconButton onClick={handleBack} aria-label={t("Back")}>
                    <ArrowLeftIcon fill={palette.text.primary} />
                </IconButton>
                <Typography variant="h6" ml={2}>{t("CreateBookmarkList")}</Typography>
            </Box>
            <Divider />
            <Box p={2} display="flex" flexDirection="column" gap={2}>
                <TextField
                    label={t("Name")}
                    value={name}
                    onChange={handleChange}
                    inputRef={nameInputRef}
                    onKeyDown={handleKeyDown}
                    fullWidth
                    disabled={isCreating}
                />
                <Box display="flex" justifyContent="flex-end">
                    <Button
                        onClick={handleCreate}
                        color="primary"
                        disabled={isCreating || name.trim() === ""}
                        aria-label={t("Create")}
                    >
                        {t("Create")}
                    </Button>
                </Box>
            </Box>
        </LargeDialog>
    );
}
