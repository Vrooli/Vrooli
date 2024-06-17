import { InputType, StandardVersion } from "@local/shared";
import { Box, Checkbox, Collapse, Container, FormControlLabel, Grid, IconButton, Tooltip, Typography, useTheme } from "@mui/material";
import { EditableText } from "components/containers/EditableText/EditableText";
import { ObjectVersionSelectSwitch } from "components/inputs/ObjectVersionSelectSwitch/ObjectVersionSelectSwitch";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { routineVersionIOInitialValues, transformRoutineVersionIOValues, validateRoutineVersionIOValues } from "forms/RoutineVersionIOForm/RoutineVersionIOForm";
import { DeleteIcon, DragIcon, ExpandLessIcon, ExpandMoreIcon } from "icons";
import { forwardRef, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { linkColors } from "styles";
import { getUserLanguages } from "utils/display/translationTools";
import { RoutineVersionInputShape } from "utils/shape/models/routineVersionInput";
import { StandardVersionShape } from "utils/shape/models/standardVersion";
import { standardInitialValues } from "views/objects/standard";
import { InputOutputListItemProps } from "../types";

type InputTypeSelect = {
    icon: React.ReactNode,
    label: string
    type: InputType | "Connect",
};

// const inputTypes: InputTypeSelect[] = [
//     { type: InputType.Text, icon: <BoldIcon />, label: "Text" },
//     { type: InputType.JSON, icon: <ObjectIcon />, label: "JSON (structured text)" },
//     { type: InputType.Dropzone, icon: <UploadIcon />, label: "Dropzone (file upload)" },
//     { type: InputType.Checkbox, icon: <BoldIcon />, label: "Checkbox (Select multiple from list)" },
//     { type: InputType.Radio, icon: <WarningIcon />, label: "Radio (Select one from list)" },
//     { type: InputType.Selector, icon: <QuoteIcon />, label: "Selector (Select one from list)" },
//     { type: InputType.Switch, icon: <TerminalIcon />, label: "Switch (Toggle on/off or true/false)" },
//     { type: InputType.IntegerInput, icon: <UnderlineIcon />, label: "Number" },
//     { type: InputType.Slider, icon: <TerminalIcon />, label: "Slider (Select a number from a range)" },
// ];

//TODO handle language change somehow
export const InputOutputListItem = forwardRef<any, InputOutputListItemProps>(({
    dragProps,
    dragHandleProps,
    index,
    isEditing,
    isInput,
    isOpen,
    item,
    handleOpen,
    handleClose,
    handleDelete,
    handleUpdate,
    language,
}, ref) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [standardVersion, setStandardVersion] = useState<StandardVersionShape>(item.standardVersion ?? standardInitialValues(session, item.standardVersion as any));
    useEffect(() => {
        setStandardVersion(item.standardVersion ?? standardInitialValues(session, item.standardVersion as any));
    }, [item, session]);

    const canUpdateStandardVersion = useMemo(() => isEditing && standardVersion.root.isInternal === true, [isEditing, standardVersion.root.isInternal]);

    const initialValues = useMemo(() => routineVersionIOInitialValues(session, isInput, item as any), [session, isInput, item]);

    const onSwitchChange = useCallback((s: StandardVersion | null) => {
        if (s && s.root.isInternal === false) {
            setStandardVersion(s as any);
        } else {
            console.log("setting standard version in inputoutputlistitem", standardInitialValues(session, item.standardVersion as any));
            setStandardVersion(standardInitialValues(session, item.standardVersion as any));
        }
    }, [item, session]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={initialValues}
            onSubmit={(values, helpers) => {
                handleUpdate(index, transformRoutineVersionIOValues(values, isInput, item, true));
                helpers.setSubmitting(false);
            }}
            validate={async (values) => await validateRoutineVersionIOValues(values, isInput, item, true)}
        >
            {(formik) => <Box
                id={`${isInput ? "input" : "output"}-item-${index}`}
                sx={{
                    zIndex: 1,
                    background: "white",
                    overflow: "hidden",
                    flexGrow: 1,
                }}
                ref={ref}
                {...dragProps}
            >
                {/* Top bar, with expand/collapse icon */}
                <Container
                    onClick={() => {
                        if (isOpen) {
                            formik.handleSubmit();
                            handleClose(index);
                        }
                        else handleOpen(index);
                    }}
                    sx={{
                        background: palette.primary.light,
                        borderBottom: `1px solid ${palette.divider}`,
                        color: "white",
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "left",
                        overflow: "hidden",
                        height: "48px", // Lighthouse SEO requirement
                        textAlign: "center",
                        cursor: "pointer",
                        paddingLeft: "8px !important",
                        paddingRight: "8px !important",
                        "&:hover": {
                            filter: "brightness(120%)",
                            transition: "filter 0.2s",
                        },
                    }}
                >
                    {/* Show order in list */}
                    <Tooltip placement="top" title="Order">
                        <Typography variant="h6" sx={{
                            margin: "0",
                            marginRight: 1,
                            padding: "0",
                        }}>
                            {index + 1}.
                        </Typography>
                    </Tooltip>
                    {/* Show delete icon if editing */}
                    {isEditing && (
                        <Tooltip placement="top" title={`Delete ${isInput ? "input" : "output"}. This will not delete the standard`}>
                            <IconButton color="inherit" onClick={() => handleDelete(index)} aria-label="delete" sx={{
                                height: "fit-content",
                                marginTop: "auto",
                                marginBottom: "auto",
                            }}>
                                <DeleteIcon fill={"white"} />
                            </IconButton>
                        </Tooltip>
                    )}
                    {/* Show name if closed */}
                    {!isOpen && (
                        <Box sx={{ display: "flex", alignItems: "center", overflow: "hidden" }}>
                            <Typography variant="h6" sx={{
                                fontWeight: "bold",
                                margin: "0",
                                padding: "0",
                                paddingRight: "0.5em",
                                overflow: "hidden",
                                textOverflow: "ellipsis",
                                whiteSpace: "nowrap",
                            }}>
                                {formik.values.name}
                            </Typography>
                        </Box>
                    )}
                    {/* Show reorder icon if editing */}
                    {isEditing && (
                        <Box {...dragHandleProps} sx={{ marginLeft: "auto" }}>
                            <DragIcon />
                        </Box>
                    )}
                    <IconButton sx={{ marginLeft: isEditing ? "unset" : "auto" }}>
                        {isOpen ?
                            <ExpandMoreIcon fill={palette.secondary.contrastText} /> :
                            <ExpandLessIcon fill={palette.secondary.contrastText} />
                        }
                    </IconButton>
                </Container>
                <Collapse in={isOpen} sx={{
                    background: palette.background.paper,
                    color: palette.background.textPrimary,
                    borderBottom: isOpen ? `1px solid ${palette.divider}` : "unset",
                }}>
                    <Grid container spacing={2} sx={{ padding: 1, ...linkColors(palette) }}>
                        <Grid item xs={12}>
                            <EditableText
                                component="TextInput"
                                isEditing={isEditing}
                                name='name'
                                props={{ label: t("Identifier"), fullWidth: true }}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <EditableText
                                component="TranslatedTextInput"
                                isEditing={isEditing}
                                name='description'
                                props={{
                                    isOptional: true,
                                    label: t("Description"),
                                    language,
                                    fullWidth: true,
                                }}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <EditableText
                                component='TranslatedMarkdown'
                                isEditing={isEditing}
                                name='helpText'
                                props={{ placeholder: t("Details"), language }}
                            />
                        </Grid>
                        {isInput && <Grid item xs={12}>
                            <Tooltip placement={"right"} title='Is this input mandatory?'>
                                <FormControlLabel
                                    disabled={!isEditing}
                                    label='This input is required'
                                    control={
                                        <Checkbox
                                            id='routine-info-dialog-is-internal'
                                            size="small"
                                            name='isRequired'
                                            color='secondary'
                                            checked={(formik.values as RoutineVersionInputShape).isRequired === true}
                                            onChange={formik.handleChange}
                                            onBlur={(e) => { formik.handleBlur(e); formik.handleSubmit(); }}
                                        />
                                    }
                                />
                            </Tooltip>
                        </Grid>}
                        {/* Select standard */}
                        <Grid item xs={12}>
                            <ObjectVersionSelectSwitch
                                canUpdate={canUpdateStandardVersion}
                                disabled={!isEditing}
                                selected={!canUpdateStandardVersion ? {
                                    translations: standardVersion.translations ?? [{ __typename: "StandardVersionTranslation" as const, language: getUserLanguages(session)[0], name: "" }],
                                } as any : null}
                                objectType="StandardVersion"
                                onChange={onSwitchChange}
                                label="Use standard"
                                tooltip="Should this be in a specific format?"
                            />
                        </Grid>
                    </Grid>
                </Collapse>
            </Box>}
        </Formik>
    );
});
InputOutputListItem.displayName = "InputOutputListItem";
