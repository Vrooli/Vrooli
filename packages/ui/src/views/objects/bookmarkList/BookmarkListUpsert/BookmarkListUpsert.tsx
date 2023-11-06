import { Bookmark, BookmarkList, BookmarkListCreateInput, BookmarkListUpdateInput, bookmarkListValidation, DUMMY_ID, endpointGetBookmarkList, endpointPostBookmarkList, endpointPutBookmarkList, noopSubmit, Session, uuid } from "@local/shared";
import { Box, Button, IconButton, List, ListItem, ListItemText, Stack, TextField, useTheme } from "@mui/material";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { AddIcon, DeleteIcon } from "icons";
import { useCallback, useContext, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { multiLineEllipsis } from "styles";
import { getDisplay } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { BookmarkShape } from "utils/shape/models/bookmark";
import { BookmarkListShape, shapeBookmarkList } from "utils/shape/models/bookmarkList";
import { validateFormValues } from "utils/validateFormValues";
import { BookmarkListFormProps, BookmarkListUpsertProps } from "../types";

const bookmarkListInitialValues = (
    session: Session | undefined,
    existing?: Partial<BookmarkList> | null | undefined,
): BookmarkListShape => ({
    __typename: "BookmarkList" as const,
    id: DUMMY_ID,
    label: "",
    bookmarks: [],
    ...existing,
});

const transformBookmarkListValues = (values: BookmarkListShape, existing: BookmarkListShape, isCreate: boolean) =>
    isCreate ? shapeBookmarkList.create(values) : shapeBookmarkList.update(existing, values);

const BookmarkListForm = ({
    disabled,
    dirty,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    ...props
}: BookmarkListFormProps) => {
    const { palette } = useTheme();
    const display = toDisplay(isOpen);
    const { t } = useTranslation();

    const [bookmarksField, , bookmarksHelpers] = useField<BookmarkShape[]>("bookmarks");

    const addNewBookmark = useCallback((to: BookmarkShape) => {
        bookmarksHelpers.setValue([...bookmarksField.value, {
            __typename: "Bookmark" as const,
            id: uuid(),
            to,
            list: { __typename: "BookmarkList", id: values.id },
        } as any]);
    }, [bookmarksField.value, bookmarksHelpers, values.id]);

    const removeBookmark = useCallback((index: number) => {
        const newBookmarks = [...bookmarksField.value];
        newBookmarks.splice(index, 1);
        bookmarksHelpers.setValue(newBookmarks);
    }, [bookmarksField, bookmarksHelpers]);

    // Search dialog to find objects to bookmark
    const hasSelectedObject = useRef(false);
    const [searchOpen, setSearchOpen] = useState(false);
    const openSearch = useCallback(() => { setSearchOpen(true); }, []);
    const closeSearch = useCallback((selectedObject?: any) => {
        setSearchOpen(false);
        hasSelectedObject.current = !!selectedObject;
        if (selectedObject) {
            addNewBookmark(selectedObject);
        }
    }, [addNewBookmark]);

    const { handleCancel, handleCompleted } = useUpsertActions<BookmarkList>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "BookmarkList",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<BookmarkList, BookmarkListCreateInput, BookmarkListUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostBookmarkList,
        endpointUpdate: endpointPutBookmarkList,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "BookmarkList" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<BookmarkListCreateInput | BookmarkListUpdateInput, BookmarkList>({
        disabled,
        existing,
        fetch,
        inputs: transformBookmarkListValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="bookmark-list-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateBookmarkList" : "UpdateBookmarkList")}
            />
            <FindObjectDialog
                find="List"
                isOpen={searchOpen}
                handleCancel={closeSearch}
                handleComplete={closeSearch}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
            >
                <Stack direction="column" spacing={2} m={2} mb={4}>
                    <Box mb={2} mt={2}>
                        <Field
                            fullWidth
                            name="label"
                            label={t("Label")}
                            as={TextField}
                        />
                    </Box>
                    {bookmarksField.value.length ? <List>
                        {bookmarksField.value.map((bookmark, index) => (
                            <ListItem
                                key={bookmark.id}
                                disablePadding
                                component={"div"}
                                sx={{
                                    display: "flex",
                                    background: palette.background.paper,
                                    padding: "8px 16px",
                                    borderBottom: `1px solid ${palette.divider}`,
                                }}
                            >
                                <Stack
                                    direction="column"
                                    spacing={1}
                                    pl={2}
                                    sx={{
                                        width: "-webkit-fill-available",
                                        display: "grid",
                                        pointerEvents: "none",
                                    }}
                                >
                                    <ListItemText
                                        primary={getDisplay(bookmark as Bookmark).title}
                                        sx={{
                                            ...multiLineEllipsis(1),
                                            lineBreak: "anywhere",
                                            pointerEvents: "none",
                                        }}
                                    />
                                    <MarkdownDisplay
                                        content={getDisplay(bookmark as Bookmark).subtitle}
                                        sx={{ ...multiLineEllipsis(2), color: palette.text.secondary, pointerEvents: "none" }}
                                    />
                                </Stack>
                                <IconButton
                                    onClick={() => removeBookmark(index)}
                                    sx={{
                                        justifySelf: "end",
                                        pointerEvents: "all",
                                    }}
                                >
                                    <DeleteIcon fill={palette.secondary.main} />
                                </IconButton>
                            </ListItem>
                        ))}
                    </List> : null}
                    <Button
                        startIcon={<AddIcon />}
                        onClick={openSearch}
                        variant="outlined"
                        sx={{ alignSelf: "center" }}
                    >
                        Add Bookmark
                    </Button>
                </Stack>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={props.errors}
                hideButtons={disabled}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />
        </MaybeLargeDialog>
    );
};

export const BookmarkListUpsert = ({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: BookmarkListUpsertProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useObjectFromUrl<BookmarkList, BookmarkListShape>({
        ...endpointGetBookmarkList,
        isCreate,
        objectType: "BookmarkList",
        overrideObject,
        transform: (data) => bookmarkListInitialValues(session, data),
    });

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformBookmarkListValues, bookmarkListValidation)}
        >
            {(formik) => <BookmarkListForm
                disabled={false}
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                {...props}
                {...formik}
            />}
        </Formik>
    );
};
