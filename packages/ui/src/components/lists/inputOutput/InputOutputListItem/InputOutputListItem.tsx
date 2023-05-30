import { DeleteIcon, DragIcon, ExpandLessIcon, ExpandMoreIcon, StandardVersion } from "@local/shared";
import { Box, Checkbox, Collapse, Container, FormControlLabel, Grid, IconButton, Tooltip, Typography, useTheme } from "@mui/material";
import { EditableText } from "components/containers/EditableText/EditableText";
import { StandardInput } from "components/inputs/standards/StandardInput/StandardInput";
import { StandardVersionSelectSwitch } from "components/inputs/StandardVersionSelectSwitch/StandardVersionSelectSwitch";
import { Formik } from "formik";
import { routineVersionIOInitialValues, transformRoutineVersionIOValues, validateRoutineVersionIOValues } from "forms/RoutineVersionIOForm/RoutineVersionIOForm";
import { standardInitialValues } from "forms/StandardForm/StandardForm";
import { forwardRef, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { linkColors } from "styles";
import { getUserLanguages } from "utils/display/translationTools";
import { SessionContext } from "utils/SessionContext";
import { RoutineVersionInputShape } from "utils/shape/models/routineVersionInput";
import { StandardVersionShape } from "utils/shape/models/standardVersion";
import { InputOutputListItemProps } from "../types";

//TODO handle language change somehow
export const InputOutputListItem = forwardRef<any, InputOutputListItemProps>(({
    dragProps,
    dragHandleProps,
    isEditing,
    index,
    isInput,
    isOpen,
    item,
    handleOpen,
    handleClose,
    handleDelete,
    handleUpdate,
    language,
    zIndex,
}, ref) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();

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
            setStandardVersion(standardInitialValues(session, item.standardVersion as any));
        }
    }, [item, session]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={initialValues}
            onSubmit={(values, helpers) => {
                handleUpdate(index, transformRoutineVersionIOValues(values, isInput, item));
                helpers.setSubmitting(false);
            }}
            validate={async (values) => await validateRoutineVersionIOValues(values, isInput, item)}
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
                        background: palette.primary.main,
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
                }}>
                    <Grid container spacing={2} sx={{ padding: 1, ...linkColors(palette) }}>
                        <Grid item xs={12}>
                            <EditableText
                                component='TextField'
                                isEditing={isEditing}
                                name='name'
                                props={{ label: "Identifier" }}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <EditableText
                                component='TranslatedTextField'
                                isEditing={isEditing}
                                name='description'
                                props={{ placeholder: "Short description (optional)", language }}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <EditableText
                                component='TranslatedMarkdown'
                                isEditing={isEditing}
                                name='helpText'
                                props={{ placeholder: "Detailed information (optional)", language, zIndex }}
                            />
                        </Grid>
                        {/* Select standard */}
                        <Grid item xs={12}>
                            <StandardVersionSelectSwitch
                                disabled={!isEditing}
                                selected={!canUpdateStandardVersion ? {
                                    translations: standardVersion.translations ?? [{ __typename: "StandardVersionTranslation" as const, language: getUserLanguages(session)[0], name: "" }],
                                } as any : null}
                                onChange={onSwitchChange}
                                zIndex={zIndex}
                            />
                        </Grid>
                        {/* Standard build/preview */}
                        <Grid item xs={12}>
                            <StandardInput
                                disabled={!canUpdateStandardVersion}
                                fieldName="preview"
                                zIndex={zIndex}
                            />
                        </Grid>
                        {isInput && <Grid item xs={12}>
                            <Tooltip placement={"right"} title='Is this input mandatory?'>
                                <FormControlLabel
                                    disabled={!isEditing}
                                    label='Required'
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
                    </Grid>
                </Collapse>
            </Box>}
        </Formik>
    );
});
