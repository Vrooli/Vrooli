/**
 * Displays all search options for a team
 */
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Checkbox from "@mui/material/Checkbox";
import Divider from "@mui/material/Divider";
import { IconButton } from "../../buttons/IconButton.js";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemButton from "@mui/material/ListItemButton";
import ListItemIcon from "@mui/material/ListItemIcon";
import ListItemText from "@mui/material/ListItemText";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material";
import { DUMMY_ID, DeleteType, endpointsActions, endpointsBookmark, endpointsBookmarkList, lowercaseFirstLetter, shapeBookmark, shapeBookmarkList, type Bookmark, type BookmarkCreateInput, type BookmarkFor, type BookmarkList, type BookmarkListCreateInput, type BookmarkSearchInput, type BookmarkSearchResult, type Count, type DeleteManyInput } from "@vrooli/shared";
import { useCallback, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { SessionContext } from "../../../contexts/session.js";
import { useLazyFetch } from "../../../hooks/useFetch.js";
import { IconCommon } from "../../../icons/Icons.js";
import { useBookmarkListsStore } from "../../../stores/bookmarkListsStore.js";
import { ELEMENT_IDS } from "../../../utils/consts.js";
import { Dialog, DialogContent, DialogActions } from "../Dialog/Dialog.js";
import { type SelectBookmarkListDialogProps } from "../types.js";

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
                    id: DUMMY_ID,
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
            <Dialog
                isOpen={isOpen}
                onClose={handleClose}
                title={
                    <Box display="flex" flexDirection="row" alignItems="center" width="100%">
                        <Typography variant="h6">{t("AddToList")}</Typography>
                        <IconButton
                            aria-label={t("AddToList")}
                            onClick={openCreate}
                            sx={createListIconButtonStyle}
                            variant="transparent"
                        >
                            <IconCommon
                                decorative
                                fill={palette.secondary.main}
                                name="Add"
                            />
                        </IconButton>
                    </Box>
                }
                size="md"
            >
                <DialogContent>
                    <List sx={bookmarkListStyle}>
                        {listItems}
                    </List>
                </DialogContent>
                <DialogActions>
                    <Button onClick={onCancel} variant="text">{t("Cancel")}</Button>
                    <Button 
                        onClick={handleSubmit} 
                        variant="contained"
                        disabled={isFindLoading || isCreateLoading || isDeleteLoading}
                    >
                        {t("Submit")}
                    </Button>
                </DialogActions>
            </Dialog>
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
                id: DUMMY_ID,
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
        <Dialog
            isOpen={isOpen}
            onClose={onClose}
            title={
                <Box display="flex" flexDirection="row" alignItems="center">
                    <IconButton
                        aria-label={t("Back")}
                        onClick={handleBack}
                        variant="transparent"
                    >
                        <IconCommon
                            decorative
                            fill={palette.text.primary}
                            name="ArrowLeft"
                        />
                    </IconButton>
                    <Typography variant="h6" ml={2}>{t("CreateBookmarkList")}</Typography>
                </Box>
            }
            size="sm"
        >
            <DialogContent>
                <Box display="flex" flexDirection="column" gap={2}>
                    <TextField
                        label={t("Name")}
                        value={name}
                        onChange={handleChange}
                        inputRef={nameInputRef}
                        onKeyDown={handleKeyDown}
                        fullWidth
                        disabled={isCreating}
                    />
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose} variant="text">{t("Cancel")}</Button>
                <Button
                    onClick={handleCreate}
                    color="primary"
                    disabled={isCreating || name.trim() === ""}
                    aria-label={t("Create")}
                    variant="contained"
                >
                    {t("Create")}
                </Button>
            </DialogActions>
        </Dialog>
    );
}
