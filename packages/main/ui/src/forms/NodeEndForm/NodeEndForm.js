import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { NodeType } from "@local/consts";
import { orDefault } from "@local/utils";
import { DUMMY_ID, uuid } from "@local/uuid";
import { nodeTranslationValidation, nodeValidation } from "@local/validation";
import { Checkbox, FormControlLabel, Stack, Tooltip } from "@mui/material";
import { useField } from "formik";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { EditableTextCollapse } from "../../components/containers/EditableTextCollapse/EditableTextCollapse";
import { combineErrorsWithTranslations, getUserLanguages } from "../../utils/display/translationTools";
import { useTranslatedFields } from "../../utils/hooks/useTranslatedFields";
import { SessionContext } from "../../utils/SessionContext";
import { validateAndGetYupErrors } from "../../utils/shape/general";
import { shapeNode } from "../../utils/shape/models/node";
import { BaseForm } from "../BaseForm/BaseForm";
export const nodeEndInitialValues = (session, routineVersion, existing) => {
    const id = uuid();
    return {
        __typename: "Node",
        id,
        end: {
            id: DUMMY_ID,
            __typename: "NodeEnd",
            node: { __typename: "Node", id },
            suggestedNextRoutineVersions: [],
            wasSuccessful: true,
        },
        nodeType: NodeType.End,
        routineVersion,
        ...existing,
        translations: orDefault(existing?.translations, [{
                __typename: "NodeTranslation",
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                name: "End",
                description: "",
            }]),
    };
};
export const transformNodeEndValues = (values, existing) => {
    return existing === undefined
        ? shapeNode.create(values)
        : shapeNode.update(existing, values);
};
export const validateNodeEndValues = async (values, existing) => {
    const transformedValues = transformNodeEndValues(values, existing);
    const validationSchema = nodeValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};
export const NodeEndForm = forwardRef(({ display, dirty, isCreate, isEditing, isLoading, isOpen, onCancel, values, zIndex, ...props }, ref) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { language, translationErrors, } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["description", "name"],
        validationSchema: nodeTranslationValidation[isCreate ? "create" : "update"]({}),
    });
    const [wasSuccessfulField] = useField("end.wasSuccessful");
    return (_jsxs(_Fragment, { children: [_jsx(BaseForm, { dirty: dirty, isLoading: isLoading, ref: ref, style: {
                    display: "block",
                    minWidth: "400px",
                    maxWidth: "700px",
                    marginBottom: "64px",
                }, children: _jsxs(Stack, { direction: "column", spacing: 4, sx: {
                        margin: 2,
                        marginBottom: 4,
                    }, children: [_jsx(EditableTextCollapse, { component: 'TranslatedTextField', isEditing: isEditing, name: "name", props: {
                                language,
                                fullWidth: true,
                                multiline: true,
                            }, title: t("Label") }), _jsx(EditableTextCollapse, { component: 'TranslatedMarkdown', isEditing: isEditing, name: "description", props: {
                                language,
                            }, title: t("Description") }), _jsx(Tooltip, { placement: "top", title: t("NodeWasSuccessfulHelp"), children: _jsx(FormControlLabel, { disabled: !isEditing, label: 'Success?', control: _jsx(Checkbox, { id: "end-node-was-successful", size: "medium", name: 'end.wasSuccessful', color: 'secondary', checked: wasSuccessfulField.value, onChange: wasSuccessfulField.onChange }) }) })] }) }), _jsx(GridSubmitButtons, { display: display, errors: combineErrorsWithTranslations(props.errors, translationErrors), isCreate: isCreate, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] }));
});
//# sourceMappingURL=NodeEndForm.js.map