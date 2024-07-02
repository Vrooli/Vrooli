import { InputType, mergeDeep, noop, noopSubmit } from "@local/shared";
import { Box, BoxProps, Divider, Grid, IconButton, List, ListItem, ListItemIcon, ListItemText, ListSubheader, Popover, Stack, Typography, styled, useTheme } from "@mui/material";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { FormInput, GeneratedGridItem, NormalizedGridContainer, normalizeFormContainers } from "components/inputs/form";
import { FormDivider } from "components/inputs/form/FormDivider/FormDivider";
import { FormHeader } from "components/inputs/form/FormHeader/FormHeader";
import { Formik } from "formik";
import { FormErrorBoundary } from "forms/FormErrorBoundary/FormErrorBoundary";
import { CreateFormInputProps, createFormInput, generateInitialValues, generateYupSchema } from "forms/generators";
import { FormBuildViewProps, FormDividerType, FormElement, FormHeaderType, FormInputType, FormRunViewProps, FormViewProps } from "forms/types";
import { usePopover } from "hooks/usePopover";
import { useWindowSize } from "hooks/useWindowSize";
import { CaseSensitiveIcon, DragIcon, Header1Icon, Header2Icon, Header3Icon, Header4Icon, HeaderIcon, LinkIcon, ListBulletIcon, ListCheckIcon, ListIcon, MinusIcon, NumberIcon, ObjectIcon, SliderIcon, SwitchIcon, CaseSensitiveIcon as TextInputIcon, UploadIcon, VrooliIcon } from "icons";
import React, { Fragment, memo, useCallback, useMemo, useRef, useState } from "react";
import { DragDropContext, Draggable, DraggableProvided, DropResult, Droppable } from "react-beautiful-dnd";
import { randomString } from "utils/codes";
import { FormStructureType } from "utils/consts";

type FormCategoryType = {
    category: string,
    items: { type: never }[],
}

/**
 * Applies restrictions to the available form element types in 
 * one of the form builder popover categories
 */
function filterCategory<T extends FormCategoryType[]>(
    category: T[0],
    limits: FormBuildViewProps["limits"],
    popover: "inputs" | "structures",
) {
    if (limits?.[popover]?.types) {
        // Filter items based on the limits.input array
        const filteredItems = category.items.filter(item => limits?.[popover]?.types?.includes(item.type));

        // If filteredItems is not empty, return category with filtered items
        if (filteredItems.length > 0) {
            return { ...category, items: filteredItems };
        }
        // If filteredItems is empty, return null
        return null;
    }
    // Otherwise, return the category as-is
    return category;
}

interface ElementBuildOuterBoxProps extends BoxProps {
    isSelected: boolean;
}

const ElementBuildOuterBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isSelected",
})<ElementBuildOuterBoxProps>(({ theme, isSelected }) => ({
    display: "flex",
    background: theme.palette.background.paper,
    border: isSelected ? `4px solid ${theme.palette.secondary.main}` : "none",
    borderRadius: "8px",
    overflow: "overlay",
    height: "100%",
    alignItems: "stretch",
}));

const ElementButton = styled("button")(() => ({
    padding: 0,
    width: "100%",
    overflow: "hidden",
    background: "none",
    color: "inherit",
    border: "none",
    textAlign: "left",
    cursor: "pointer",
}));

const DragBox = styled(Box)(({ theme }) => ({
    cursor: "pointer",
    display: "flex",
    flexDirection: "column",
    padding: "4px",
    background: theme.palette.secondary.main,
}));

const dragIconStyle = { marginTop: "auto", marginBottom: "auto" } as const;

interface ToolbarBoxProps extends BoxProps {
    formElementsCount: number;
}

const ToolbarBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "formElementsCount",
})<ToolbarBoxProps>(({ theme, formElementsCount }) => ({
    display: "flex",
    justifyContent: "space-around",
    margin: "-4px",
    padding: "4px",
    background: theme.palette.primary.main,
    color: theme.palette.secondary.contrastText,
    borderBottomLeftRadius: formElementsCount === 0 ? "8px" : "0px",
    borderBottomRightRadius: formElementsCount === 0 ? "8px" : "0px",
}));

const toolbarLargeButtonStyle = { cursor: "pointer" } as const;

const popoverAnchorOrigin = { vertical: "bottom", horizontal: "center" } as const;

interface PopoverListItemProps {
    icon: React.ReactNode;
    label: string;
    tag?: FormHeaderType["tag"];
    type: FormElement["type"];
    onAddDivider: () => unknown;
    onAddHeader: (headerData: Partial<FormHeaderType>) => unknown;
    onAddInput: (InputData: Omit<Partial<FormInputType>, "type"> & { type: InputType; }) => unknown;
}

const PopoverListItem = memo(function PopoverListItemMemo({
    icon,
    label,
    onAddDivider,
    onAddHeader,
    onAddInput,
    tag,
    type,
}: PopoverListItemProps) {
    const handleClick = useCallback(() => {
        if (type === "Divider") {
            onAddDivider();
        } else if (type === "Header") {
            if (tag) {
                onAddHeader({ tag });
            } else {
                console.error("Missing tag for header - cannot add header to form.");
            }
        } else {
            onAddInput({ type });
        }
    }, [onAddDivider, onAddHeader, onAddInput, tag, type]);

    return (
        <ListItem
            button
            onClick={handleClick}
        >
            <ListItemIcon>{icon}</ListItemIcon>
            <ListItemText primary={label} />
        </ListItem>
    );
});

const FormHelperText = styled(Typography)(({ theme }) => ({
    textAlign: "center",
    color: theme.palette.text.secondary,
    padding: "20px",
}));

const formDividerStyle = { marginBottom: 2 } as const;

//TODO: Allow titles to collapse sections below them. Need to update the way inputs are rendered so that they're nested within the header using a collapsibetext component
//TODO pass in state as props and add callback to update
//TODO create FormView component which switches between build and run mode, and hook to handle creating and passing in form state
export function FormBuildView({
    limits,
    onSchemaChange,
    schema,
}: FormBuildViewProps) {
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width < breakpoints.values.sm);

    const [selectedElementIndex, setSelectedElementIndex] = useState<number | null>(null);
    const elementRefs = useRef<(HTMLElement | null)[]>([]);

    const [inputAnchorEl, openInputPopover, closeInputPopover, isInputPopoverOpen] = usePopover();
    const handleInputPopoverOpen = useCallback(function handleInputPopoverOpenCallback(event: React.MouseEvent<HTMLElement>) {
        event.preventDefault();
        openInputPopover(event);
    }, [openInputPopover]);

    const [structureAnchorEl, openStructurePopover, closeStructurePopover, isStructurePopoverOpen] = usePopover();
    const handleStructurePopoverOpen = useCallback(function handleStructurePopoverOpenCallback(event: React.MouseEvent<HTMLElement>) {
        event.preventDefault();
        openStructurePopover(event);
    }, [openStructurePopover]);

    const closePopovers = useCallback(function closePopoversCallback() {
        closeInputPopover();
        closeStructurePopover();
    }, [closeStructurePopover, closeInputPopover]);

    const handleAddElement = useCallback(function handleAddElementCallback<T extends FormElement>(element: Omit<T, "id">) {
        const newElement = {
            ...element,
            id: Date.now().toString(),
        } as unknown as FormElement;
        const newElements = [...schema.elements];
        if (selectedElementIndex !== null) {
            newElements.splice(selectedElementIndex + 1, 0, newElement);
        } else {
            newElements.push(newElement);
        }
        onSchemaChange({ ...schema, elements: newElements });
        setSelectedElementIndex(newElements.length - 1);
    }, [onSchemaChange, schema, selectedElementIndex]);

    const handleDeleteElement = useCallback(function handleDeleteElementCallback(index: number) {
        const newElements = [...schema.elements];
        newElements.splice(index, 1);
        onSchemaChange({ ...schema, elements: newElements });
        // Set the selection to the next element if there is one
        if (index < newElements.length) {
            setSelectedElementIndex(index);
        } else if (index > 0) {
            setSelectedElementIndex(index - 1);
        } else {
            setSelectedElementIndex(null);
        }
    }, [schema, onSchemaChange]);

    function selectElement(index: number) {
        setSelectedElementIndex(index);
    }

    const inputItems = useMemo(function inputItemsMemo() {
        return ([
            {
                category: "Text Inputs",
                items: [
                    { type: InputType.Text, icon: <CaseSensitiveIcon />, label: "Text" },
                    { type: InputType.JSON, icon: <ObjectIcon />, label: "JSON (structured text)" }, //TODO
                ],
            },
            {
                category: "Selection Inputs",
                items: [
                    { type: InputType.Checkbox, icon: <ListCheckIcon />, label: "Checkbox (Select multiple from list)" },
                    { type: InputType.Radio, icon: <ListBulletIcon />, label: "Radio (Select one from list)" },
                    { type: InputType.Selector, icon: <ListIcon />, label: "Selector (Select one from list)" }, // TODO waiting on InputType.JSON to define item shape (standard) and InputType.LinkItem to connect standard
                    { type: InputType.Switch, icon: <SwitchIcon />, label: "Switch (Toggle on/off or true/false)" },
                ],
            },
            {
                category: "Link Inputs",
                items: [
                    { type: InputType.LinkUrl, icon: <LinkIcon />, label: "Link any URL" }, //TODO
                    { type: InputType.LinkItem, icon: <VrooliIcon />, label: "Link Vrooli object" }, //TODO
                ],
            },
            {
                category: "Numeric Inputs",
                items: [
                    { type: InputType.IntegerInput, icon: <NumberIcon />, label: "Number" },
                    { type: InputType.Slider, icon: <SliderIcon />, label: "Slider (Select a number from a range)" },
                ],
            },
            {
                category: "File Inputs",
                items: [
                    { type: InputType.Dropzone, icon: <UploadIcon />, label: "Dropzone (file upload)" },
                ],
            },
        ] as const).reduce((acc, category) => {
            const filteredCategory = filterCategory<typeof inputItems>(category, limits, "inputs");
            if (filteredCategory) {
                acc.push(filteredCategory);
            }
            return acc;
        }, [] as typeof inputItems);
    }, [limits]);

    const structureItems = useMemo(function structureItemsMemo() {
        return ([
            {
                category: "Headers",
                items: [
                    { type: "Header", tag: "h1", icon: <Header1Icon />, label: "Title (Largest)" },
                    { type: "Header", tag: "h2", icon: <Header2Icon />, label: "Subtitle (Large)" },
                    { type: "Header", tag: "h3", icon: <Header3Icon />, label: "Header (Medium)" },
                    { type: "Header", tag: "h4", icon: <Header4Icon />, label: "Subheader (Small)" },
                ],
            }, {
                category: "Page Elements",
                items: [
                    { type: "Divider", icon: <MinusIcon />, label: "Divider" },
                ],
            },
        ] as const).reduce((acc, category) => {
            const filteredCategory = filterCategory<typeof inputItems>(category, limits, "structures");
            if (filteredCategory) {
                acc.push(filteredCategory);
            }
            return acc;
        }, [] as typeof inputItems);
    }, [limits]);

    const handleAddInput = useCallback(function handleAddInputCallback(data: Omit<Partial<FormInputType>, "type"> & { type: InputType }) {
        const newElement = createFormInput({
            fieldName: `input-${randomString()}`,
            label: `Input #${selectedElementIndex !== null ? selectedElementIndex + 1 : schema.elements.length + 1}`,
            ...data,
        } as CreateFormInputProps);
        if (!newElement) {
            console.error("Failed to create form input", data);
            return;
        }
        handleAddElement(newElement);
        closeInputPopover();
    }, [handleAddElement, closeInputPopover, selectedElementIndex, schema.elements.length]);

    const handleUpdateInput = useCallback(function handleUpdateInputCallback(index: number, data: Partial<FormInputType>) {
        const element = mergeDeep(data, schema.elements[index] as FormInputType);
        const newElements = [...schema.elements.slice(0, index), element, ...schema.elements.slice(index + 1)] as FormElement[];
        onSchemaChange({ ...schema, elements: newElements });
    }, [onSchemaChange, schema]);

    const handleAddHeader = useCallback(function handleAddHeaderCallback(data: Partial<FormHeaderType>) {
        const tag = data.tag ?? "h1";
        handleAddElement<FormHeaderType>({
            type: FormStructureType.Header,
            label: data.label ?? `New ${tag.toUpperCase()}`,
            tag,
            ...data,
        });
        closeStructurePopover();
    }, [handleAddElement, closeStructurePopover]);

    const handleUpdateHeader = useCallback(function handleUpdateHeaderCallback(index: number, data: Partial<FormHeaderType>) {
        const element = {
            ...schema.elements[index],
            ...data,
        };
        const newElements = [...schema.elements.slice(0, index), element, ...schema.elements.slice(index + 1)] as FormElement[];
        onSchemaChange({ ...schema, elements: newElements });
    }, [onSchemaChange, schema]);

    const handleAddDivider = useCallback(function handleAddDividerCallback() {
        handleAddElement<FormDividerType>({
            type: FormStructureType.Divider,
            label: "",
        });
        closeStructurePopover();
    }, [handleAddElement, closeStructurePopover]);

    const onDragEnd = useCallback((result: DropResult) => {
        const { source, destination } = result;
        if (!destination || result.type !== "formElement") return;
        const newElements = [...schema.elements];
        const [removed] = newElements.splice(source.index, 1);
        newElements.splice(destination.index, 0, removed);
        onSchemaChange({ ...schema, elements: newElements });
        setSelectedElementIndex(destination.index);
    }, [onSchemaChange, schema]);

    function renderElement(element: FormElement, index: number, providedDrag: DraggableProvided) {
        const isSelected = selectedElementIndex === index;

        function handleElementButtonRef(ref: HTMLElement | null) {
            elementRefs.current[index] = ref;
        }
        function onElementButtonClick() {
            selectElement(index);
        }
        function onElementButtonKeyDown(e: React.KeyboardEvent) {
            if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                selectElement(index);
            }
        }

        function onFormHeaderUpdate(data: Partial<FormHeaderType>) {
            if (!isSelected) return;
            handleUpdateHeader(index, data);
        }
        function onFormInputConfigUpdate(updatedInput: Partial<FormInputType>) {
            if (!isSelected) return;
            handleUpdateInput(index, updatedInput);
        }
        function onFormElementDelete() {
            if (!isSelected) return;
            handleDeleteElement(index);
        }

        return (
            <ElementBuildOuterBox
                isSelected={isSelected}
                key={element.id}
                ref={providedDrag.innerRef}
                {...providedDrag.draggableProps}
            >
                <ElementButton
                    ref={handleElementButtonRef}
                    onClick={onElementButtonClick}
                    onKeyDown={onElementButtonKeyDown}
                    type="button"
                >
                    {element.type === FormStructureType.Header && (
                        <FormHeader
                            element={element as FormHeaderType}
                            isEditing={isSelected}
                            onUpdate={onFormHeaderUpdate}
                            onDelete={onFormElementDelete}
                        />
                    )}
                    {element.type === FormStructureType.Divider && (
                        <FormDivider
                            isEditing={isSelected}
                            onDelete={onFormElementDelete}
                        />
                    )}
                    {!(element.type in FormStructureType) && (
                        <FormInput
                            fieldData={element as FormInputType}
                            index={index}
                            isEditing={isSelected}
                            onConfigUpdate={onFormInputConfigUpdate}
                            onDelete={onFormElementDelete}
                        />)}
                    {isSelected ? Toolbar : null}
                </ElementButton>
                {isSelected && (
                    <DragBox
                        {...providedDrag.dragHandleProps}
                    >
                        <DragIcon
                            fill={palette.secondary.contrastText}
                            width="24px"
                            height="24px"
                            style={dragIconStyle}
                        />
                    </DragBox>
                )}
            </ElementBuildOuterBox>
        );
    }

    const Toolbar = useMemo(() => {
        const numInputItems = inputItems.reduce((acc, { items }) => acc + items.length, 0);
        const firstInputItem = numInputItems === 1 ? inputItems[0].items[0] : null;
        const DisplayedInputIcon = firstInputItem ? (() => firstInputItem.icon) : TextInputIcon;
        function inputOnClick(event: React.MouseEvent<HTMLElement>) {
            if (firstInputItem) {
                handleAddInput({ type: firstInputItem.type });
            } else {
                handleInputPopoverOpen(event);
            }
        }

        const numStructureItems = structureItems.length;
        const firstStructureItem = numStructureItems === 1 ? structureItems[0].items[0] : null;
        const DisplayedStructureIcon = firstStructureItem ? (() => firstStructureItem.icon) : HeaderIcon;
        function structureOnClick(event: React.MouseEvent<HTMLElement>) {
            if (firstStructureItem) {
                if (firstStructureItem.type === "Divider") {
                    handleAddDivider();
                } else {
                    handleAddHeader({ tag: firstStructureItem.tag });
                }
            } else {
                handleStructurePopoverOpen(event);
            }
        }

        return (
            <ToolbarBox formElementsCount={schema.elements.length}>
                {numInputItems > 0 && <>
                    {isMobile ? <IconButton onClick={inputOnClick}>
                        <DisplayedInputIcon width={24} height={24} />
                    </IconButton> : <Typography variant="body1" sx={toolbarLargeButtonStyle} onClick={inputOnClick}>
                        {numInputItems === 1 ? `Add ${inputItems[0].items[0].label.toLowerCase()}` : "Add input"}
                    </Typography>}
                </>}
                {numStructureItems > 0 && <>
                    {isMobile ? <IconButton onClick={structureOnClick}>
                        <DisplayedStructureIcon width={24} height={24} />
                    </IconButton> : <Typography variant="body1" sx={toolbarLargeButtonStyle} onClick={structureOnClick}>
                        {numStructureItems === 1 ? `Add ${structureItems[0].label.toLowerCase()}` : "Add structure"}
                    </Typography>}
                </>}
            </ToolbarBox>
        );
    }, [inputItems, structureItems, schema.elements.length, isMobile, handleAddInput, handleInputPopoverOpen, handleAddDivider, handleAddHeader, handleStructurePopoverOpen]);

    //TODO calculate using defaultValues
    const formInitialValues = useMemo(() => ({}) as Record<string, unknown>, []);

    return (
        <div>
            <Popover
                open={isInputPopoverOpen}
                anchorEl={inputAnchorEl}
                onClose={closeInputPopover}
                anchorOrigin={popoverAnchorOrigin}
            >

                {inputItems.map(({ category, items }) => (
                    <Fragment key={category}>
                        <ListSubheader>{category}</ListSubheader>
                        {items.map((item) => (
                            <PopoverListItem
                                key={item.tag}
                                icon={item.icon}
                                label={item.label}
                                tag={item.tag}
                                type={item.type}
                                onAddDivider={handleAddDivider}
                                onAddHeader={handleAddHeader}
                                onAddInput={handleAddInput}
                            />
                        ))}
                        <Divider />
                    </Fragment>
                ))}
            </Popover>
            <Popover
                open={isStructurePopoverOpen}
                anchorEl={structureAnchorEl}
                onClose={closeStructurePopover}
                anchorOrigin={popoverAnchorOrigin}
            >
                <List>
                    {structureItems.map(({ category, items }) => (
                        <Fragment key={category}>
                            <ListSubheader>{category}</ListSubheader>
                            {items.map((item) => (
                                <PopoverListItem
                                    key={item.tag}
                                    icon={item.icon}
                                    label={item.label}
                                    tag={item.tag}
                                    type={item.type}
                                    onAddDivider={handleAddDivider}
                                    onAddHeader={handleAddHeader}
                                    onAddInput={handleAddInput}
                                />
                            ))}
                            <Divider />
                        </Fragment>
                    ))}
                </List>
            </Popover>
            {schema.elements.length === 0 && <FormHelperText variant="body1">Use the options below to populate the form.</FormHelperText>}
            {schema.elements.length === 0 ? Toolbar : null}
            {schema.elements.length > 0 && <Divider sx={formDividerStyle} />}
            {/* Formik to handle form state. Even though we're not entering data, we need to make sure we aren't using a formik context higher up in the tree */}
            <Formik
                enableReinitialize={true}
                initialValues={formInitialValues}
                onSubmit={noopSubmit}
            >
                {() => (
                    <FormErrorBoundary onError={closePopovers}> {/* Error boundary to catch elements that fail to render */}
                        {/* Drag/Drop context to reorder elements */}
                        <DragDropContext onDragEnd={onDragEnd}>
                            <Droppable droppableId="form-builder-elements" direction="vertical" type="formElement">
                                {(providedDrop) => (
                                    <Box ref={providedDrop.innerRef} {...providedDrop.droppableProps}> {/* Droppable area for form elements */}
                                        {schema.elements.map((element, index) => (
                                            <Draggable key={`${element.id}-drag`} draggableId={`${element.id}-drag`} index={index}>
                                                {(providedDrag) => renderElement(element, index, providedDrag)}
                                            </Draggable>
                                        ))}
                                        {providedDrop.placeholder}
                                    </Box>
                                )}
                            </Droppable>
                        </DragDropContext>
                    </FormErrorBoundary>
                )}
            </Formik>
        </div>
    );
}

const ElementRunOuterBox = styled("button")(() => ({
    padding: 0,
    width: "100%",
    overflow: "hidden",
    background: "none",
    color: "inherit",
    border: "none",
    textAlign: "left",
    cursor: "pointer",
}));

export function FormRunView({
    disabled,
    schema,
}: FormRunViewProps) {

    const renderElement = useCallback(function renderElementMemo(element: FormElement, index: number) {
        return (
            <ElementRunOuterBox
                key={element.id}
            >
                {element.type === FormStructureType.Header && (
                    <FormHeader
                        element={element as FormHeaderType}
                        isEditing={false}
                        onDelete={noop}
                        onUpdate={noop}
                    />
                )}
                {element.type === FormStructureType.Divider && (
                    <FormDivider
                        isEditing={false}
                        onDelete={noop}
                    />
                )}
                {!(element.type in FormStructureType) && (
                    <FormInput
                        disabled={disabled}
                        fieldData={element as FormInputType}
                        index={index}
                        isEditing={false}
                        onConfigUpdate={noop}
                        onDelete={noop}
                    />)}
            </ElementRunOuterBox>
        );
    }, [disabled]);

    const sections = useMemo(() => {
        // Normalize/heal containers to ensure they cover all elements
        const normalizedContainers = normalizeFormContainers(schema);
        // Render each container as a stack or grid, depending on configuration
        const sections: JSX.Element[] = [];
        for (let i = 0; i < normalizedContainers.length; i++) {
            const currContainer: NormalizedGridContainer = normalizedContainers[i];
            const containerProps = {
                direction: currContainer?.direction ?? "column",
                key: `form-section-container-${i}`,
                spacing: 2,
            };
            // Use grid for horizontal layout, and stack for vertical layout
            const useGrid = containerProps.direction === "row";
            // Generate component for each field in the grid
            const gridItems: JSX.Element[] = [];
            for (let j = currContainer.startIndex; j <= currContainer.endIndex; j++) {
                const fieldData = schema.elements[j] as FormInputType;
                gridItems.push(<GeneratedGridItem
                    key={`grid-item-${fieldData.id}`}
                    fieldsInGrid={currContainer.endIndex - currContainer.startIndex + 1}
                    isInGrid={useGrid}
                >
                    {renderElement(fieldData, j)}
                </GeneratedGridItem>);
            }
            const itemsContainer = useGrid ? <Grid container {...containerProps}>
                {gridItems}
            </Grid> : <Stack {...containerProps}>
                {gridItems}
            </Stack>;
            // Each section is represented using a fieldset
            sections.push(
                <fieldset key={`grid-container-${i}`} style={{ border: "none" }}>
                    {/* If a title is provided, the items are wrapped in a collapsible container */}
                    {currContainer?.title ? <ContentCollapse
                        disableCollapse={currContainer.disableCollapse}
                        helpText={currContainer.description ?? undefined}
                        title={currContainer.title}
                        titleComponent="legend"
                    >
                        {itemsContainer}
                    </ContentCollapse> : itemsContainer}
                </fieldset>,
            );
        }
        return sections;
    }, [renderElement, schema]);

    const initialValues = useMemo(function initialValuesMemo() {
        return generateInitialValues(schema.elements);
    }, [schema]);
    const validationSchema = useMemo(function validationSchemaMemo() {
        return generateYupSchema(schema);
    }, [schema]);

    return (
        <div>
            {schema.elements.length === 0 && <FormHelperText variant="body1">The form is empty.</FormHelperText>}
            {schema.elements.length > 0 && <Divider sx={formDividerStyle} />}
            {/* Formik to handle form state. Even though we're not entering data, we need to make sure we aren't using a formik context higher up in the tree */}
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={noopSubmit} //TODO
                validationSchema={validationSchema}
            >
                {() => (
                    <FormErrorBoundary> {/* Error boundary to catch elements that fail to render */}
                        <Stack
                            direction={"column"}
                            key={"form-container"}
                            spacing={4}
                            sx={{ width: "100%" }}
                        >
                            {sections}
                        </Stack>
                    </FormErrorBoundary>
                )}
            </Formik>
        </div>
    );
}

export function FormView({
    isEditing,
    ...props
}: FormViewProps) {
    if (isEditing) {
        return <FormBuildView {...props} />;
    }
    return <FormRunView {...props} />;
}
