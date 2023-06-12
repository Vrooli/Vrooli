import { BookmarkList, bookmarkListValidation, DUMMY_ID, Session } from "@local/shared";
import { Box, TextField } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { Field } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { BookmarkListFormProps } from "forms/types";
import { forwardRef } from "react";
import { useTranslation } from "react-i18next";
import { validateAndGetYupErrors } from "utils/shape/general";
import { BookmarkListShape, shapeBookmarkList } from "utils/shape/models/bookmarkList";

export const bookmarkListInitialValues = (
    session: Session | undefined,
    existing?: BookmarkList | null | undefined,
): BookmarkListShape => ({
    __typename: "BookmarkList" as const,
    id: DUMMY_ID,
    label: "",
    bookmarks: [],
    ...existing,
});

export function transformBookmarkListValues(values: BookmarkListShape, existing?: BookmarkListShape) {
    console.log("transformBookmarkListValues", values, shapeBookmarkList.create(values));
    return existing === undefined
        ? shapeBookmarkList.create(values)
        : shapeBookmarkList.update(existing, values);
}

export const validateBookmarkListValues = async (values: BookmarkListShape, existing?: BookmarkListShape) => {
    const transformedValues = transformBookmarkListValues(values, existing);
    const validationSchema = bookmarkListValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const BookmarkListForm = forwardRef<any, BookmarkListFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    zIndex,
    ...props
}, ref) => {
    const { t } = useTranslation();
    console.log("bookmark list errorrrrrr", props.errors);

    return (
        <>
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
                maxWidth={700}
                ref={ref}
            >
                <Box mb={2} mt={2}>
                    <Field
                        fullWidth
                        name="label"
                        label={t("Label")}
                        as={TextField}
                    />
                </Box>
                <GridSubmitButtons
                    display={display}
                    errors={props.errors}
                    isCreate={isCreate}
                    loading={props.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={props.handleSubmit}
                />
            </BaseForm>
        </>
    );
});
