import { Bookmark, BookmarkList, BookmarkListCreateInput, BookmarkListUpdateInput, bookmarkListValidation, DUMMY_ID, endpointGetBookmarkList, endpointPostBookmarkList, endpointPutBookmarkList, noopSubmit, Session, uuid } from "@local/shared";
import { Box, Button, IconButton, List, ListItem, ListItemText, Stack, TextField, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { BookmarkListFormProps } from "forms/types";
import { useConfirmBeforeLeave } from "hooks/useConfirmBeforeLeave";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { AddIcon, DeleteIcon } from "icons";
import { useCallback, useContext, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { multiLineEllipsis } from "styles";
import { getDisplay } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { validateAndGetYupErrors } from "utils/shape/general";
import { BookmarkShape } from "utils/shape/models/bookmark";
import { BookmarkListShape, shapeBookmarkList } from "utils/shape/models/bookmarkList";
import { BookmarkListUpsertProps } from "../types";

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

const validateBookmarkListValues = async (values: BookmarkListShape, existing: BookmarkListShape, isCreate: boolean) => {
    const transformedValues = transformBookmarkListValues(values, existing, isCreate);
    const validationSchema = bookmarkListValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" });
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

const BookmarkListForm = ({
    disabled,
    dirty,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onCancel,
    onCompleted,
    onDeleted,
    values,
    ...props
}: BookmarkListFormProps) => {
    const { palette } = useTheme();
    const display = toDisplay(isOpen);
    const { t } = useTranslation();

    const [idField, , idHelpers] = useField<string>("id");
    const [bookmarksField, bookmarksMeta, bookmarksHelpers] = useField<BookmarkShape[]>("bookmarks");

    const addNewBookmark = useCallback((to: BookmarkShape) => {
        bookmarksHelpers.setValue([...bookmarksField.value, {
            __typename: "Bookmark" as const,
            id: uuid(),
            to,
            list: { __typename: "BookmarkList", id: idField.value },
        } as any]);
    }, [bookmarksField, bookmarksHelpers, idField]);

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

    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<BookmarkList, BookmarkListCreateInput, BookmarkListUpdateInput>({
        display,
        endpointCreate: endpointPostBookmarkList,
        endpointUpdate: endpointPutBookmarkList,
        isCreate,
        onCancel,
        onCompleted,
    });
    const { handleClose } = useConfirmBeforeLeave({ handleCancel, shouldPrompt: dirty });
    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useCallback(() => {
        if (disabled) {
            PubSub.get().publishSnack({ messageKey: "Unauthorized", severity: "Error" });
            return;
        }
        if (!isCreate && !existing) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        fetchLazyWrapper<BookmarkListCreateInput | BookmarkListUpdateInput, BookmarkList>({
            fetch,
            inputs: transformBookmarkListValues(values, existing, isCreate),
            onSuccess: (data) => { handleCompleted(data); },
            onCompleted: () => { props.setSubmitting(false); },
        });
    }, [disabled, existing, fetch, handleCompleted, isCreate, props, values]);

    return (
        <MaybeLargeDialog
            display={display}
            id="bookmark-list-upsert-dialog"
            isOpen={isOpen}
            onClose={handleClose}
        >
            <TopBar
                display={display}
                onClose={handleClose}
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
                errors={props.errors as any}
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
        objectType: "BookmarkList",
        overrideObject,
        transform: (data) => bookmarkListInitialValues(session, data),
    });

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateBookmarkListValues(values, existing, isCreate)}
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
