import { InputType, mergeDeep, noopSubmit } from "@local/shared";
import { Box, BoxProps, Divider, IconButton, List, ListItem, ListItemIcon, ListItemText, ListSubheader, Popover, Typography, styled, useTheme } from "@mui/material";
import { FormInput } from "components/inputs/form";
import { FormHeader } from "components/inputs/form/FormHeader/FormHeader";
import { Formik } from "formik";
import { FormErrorBoundary } from "forms/FormErrorBoundary/FormErrorBoundary";
import { CreateFormInputProps, createFormInput } from "forms/generators";
import { FormBuildViewProps, FormElement, FormHeaderType, FormInputType } from "forms/types";
import { usePopover } from "hooks/usePopover";
import { useWindowSize } from "hooks/useWindowSize";
import { CaseSensitiveIcon, DragIcon, Header1Icon, Header2Icon, Header3Icon, Header4Icon, HeaderIcon, LinkIcon, ListBulletIcon, ListCheckIcon, ListIcon, NumberIcon, ObjectIcon, SliderIcon, SwitchIcon, CaseSensitiveIcon as TextInputIcon, UploadIcon, VrooliIcon } from "icons";
import React, { Fragment, memo, useCallback, useMemo, useRef, useState } from "react";
import { DragDropContext, Draggable, DraggableProvided, DropResult, Droppable } from "react-beautiful-dnd";
import { randomString } from "utils/codes";

interface ElementOuterBoxProps extends BoxProps {
    isSelected: boolean;
}

const ElementOuterBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isSelected",
})<ElementOuterBoxProps>(({ theme, isSelected }) => ({
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

interface HeaderListItemProps {
    icon: React.ReactNode;
    label: string;
    tag: FormHeaderType["tag"];
    onAddHeader: (headerData: Partial<FormHeaderType>) => unknown;
}

const HeaderListItem = memo(function HeaderListItemMemo({ icon, label, tag, onAddHeader }: HeaderListItemProps) {
    const handleClick = useCallback(() => {
        onAddHeader({ tag });
    }, [onAddHeader, tag]);

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

interface InputListItemProps {
    icon: React.ReactNode;
    label: string;
    type: InputType;
    onAddInput: (InputData: Omit<Partial<FormInputType>, "type"> & { type: InputType; }) => unknown;
}

const InputListItem = memo(function InputListItemMemo({ icon, label, type, onAddInput }: InputListItemProps) {
    const handleClick = useCallback(() => {
        onAddInput({ type });
    }, [onAddInput, type]);

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

//TODO: Add groups and page breaks
export function FormBuildView({
    limits,
}: FormBuildViewProps) {
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width < breakpoints.values.sm);

    const [formElements, setFormElements] = useState<FormElement[]>([]);
    const [selectedElementIndex, setSelectedElementIndex] = useState<number | null>(null);
    const elementRefs = useRef<(HTMLElement | null)[]>([]);

    const [headerAnchorEl, openHeaderPopover, closeHeaderPopover, isHeaderPopoverOpen] = usePopover();
    const handleHeaderPopoverOpen = useCallback(function handleHeaderPopoverOpenCallback(event: React.MouseEvent<HTMLElement>) {
        event.preventDefault();
        openHeaderPopover(event);
    }, [openHeaderPopover]);

    const [inputAnchorEl, openInputPopover, closeInputPopover, isInputPopoverOpen] = usePopover();
    const handleInputPopoverOpen = useCallback(function handleInputPopoverOpenCallback(event: React.MouseEvent<HTMLElement>) {
        event.preventDefault();
        openInputPopover(event);
    }, [openInputPopover]);

    const closePopovers = useCallback(function closePopoversCallback() {
        closeHeaderPopover();
        closeInputPopover();
    }, [closeHeaderPopover, closeInputPopover]);

    const handleAddElement = useCallback(function handleAddElementCallback<T extends FormElement>(element: Omit<T, "id">) {
        const newElement = {
            ...element,
            id: Date.now().toString(),
        } as unknown as FormElement;
        const newElements = [...formElements];
        if (selectedElementIndex !== null) {
            newElements.splice(selectedElementIndex + 1, 0, newElement);
        } else {
            newElements.push(newElement);
        }
        setFormElements(newElements);
        setSelectedElementIndex(newElements.length - 1);
    }, [formElements, selectedElementIndex]);

    function handleDeleteElement(index: number) {
        setFormElements((elements) => {
            const newElements = [...elements];
            newElements.splice(index, 1);
            // Set the selection to the next element if there is one
            if (index < newElements.length) {
                setSelectedElementIndex(index);
            } else if (index > 0) {
                setSelectedElementIndex(index - 1);
            } else {
                setSelectedElementIndex(null);
            }
            return newElements;
        });
    }

    function selectElement(index: number) {
        setSelectedElementIndex(index);
    }

    const headerItems = useMemo(function headerItemsMemo() {
        return ([
            { tag: "h1", icon: <Header1Icon />, label: "Title (Largest)" },
            { tag: "h2", icon: <Header2Icon />, label: "Subtitle (Large)" },
            { tag: "h3", icon: <Header3Icon />, label: "Header (Medium)" },
            { tag: "h4", icon: <Header4Icon />, label: "Subheader (Small)" },
        ] as const).filter(({ tag }) => {
            if (limits?.headers?.types) {
                return limits.headers.types.includes(tag);
            }
            return true;
        });
    }, [limits?.headers?.types]);

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
            if (limits?.inputs?.types) {
                // Filter items based on the limits.input array
                const filteredItems = category.items.filter(item => limits?.inputs?.types?.includes(item.type));

                // If filteredItems is not empty, push the category with these items into the accumulator
                if (filteredItems.length > 0) {
                    acc.push({ ...category, items: filteredItems });
                }
            } else {
                // If no limits are set, push the category as is
                acc.push(category);
            }
            return acc; // Return the updated accumulator for the next iteration
        }, [] as typeof inputItems);
    }, [limits?.inputs?.types]);


    const handleAddHeader = useCallback(function handleAddHeaderCallback(data: Partial<FormHeaderType>) {
        const tag = data.tag ?? "h1";
        handleAddElement<FormHeaderType>({
            type: "Header",
            label: data.label ?? `New ${tag.toUpperCase()}`,
            tag,
            ...data,
        });
        closeHeaderPopover();
    }, [handleAddElement, closeHeaderPopover]);

    const handleUpdateHeader = useCallback(function handleUpdateHeaderCallback(index: number, data: Partial<FormHeaderType>) {
        const element = {
            ...formElements[index],
            ...data,
        };
        const newElements = [...formElements.slice(0, index), element, ...formElements.slice(index + 1)] as FormElement[];
        console.log("in handleUpdateHeader", newElements);
        setFormElements(newElements);
    }, [formElements]);

    const handleAddInput = useCallback(function handleAddInputCallback(data: Omit<Partial<FormInputType>, "type"> & { type: InputType }) {
        const newElement = createFormInput({
            fieldName: `input-${randomString()}`,
            label: `Input #${selectedElementIndex !== null ? selectedElementIndex + 1 : formElements.length + 1}`,
            ...data,
        } as CreateFormInputProps);
        if (!newElement) {
            console.error("Failed to create form input", data);
            return;
        }
        handleAddElement(newElement);
        closeInputPopover();
    }, [handleAddElement, closeInputPopover, selectedElementIndex, formElements.length]);

    const handleUpdateInput = useCallback(function handleUpdateInputCallback(index: number, data: Partial<FormInputType>) {
        const element = mergeDeep(data, formElements[index] as FormInputType);
        const newElements = [...formElements.slice(0, index), element, ...formElements.slice(index + 1)] as FormElement[];
        setFormElements(newElements);
    }, [formElements]);

    const onDragEnd = useCallback((result: DropResult) => {
        const { source, destination } = result;
        if (!destination || result.type !== "formElement") return;
        // Update order of elements in form
        setFormElements((formElements) => {
            const newElements = [...formElements];
            const [removed] = newElements.splice(source.index, 1);
            newElements.splice(destination.index, 0, removed);
            // Also update the selected element index
            setSelectedElementIndex(destination.index);
            return newElements;
        });
    }, []);

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
        function onFormHeaderDelete() {
            if (!isSelected) return;
            handleDeleteElement(index);
        }

        function onFormInputConfigUpdate(updatedInput: Partial<FormInputType>) {
            if (!isSelected) return;
            handleUpdateInput(index, updatedInput);
        }
        function onFormInputDelete() {
            if (!isSelected) return;
            handleDeleteElement(index);
        }

        return (
            <ElementOuterBox
                isSelected={isSelected}
                key={element.id}
                ref={providedDrag.innerRef}
                {...providedDrag.draggableProps}
            >
                <ElementButton
                    ref={handleElementButtonRef}
                    onClick={onElementButtonClick}
                    onKeyDown={onElementButtonKeyDown}
                >
                    {element.type === "Header" && (
                        <FormHeader
                            element={element as FormHeaderType}
                            isSelected={isSelected}
                            onUpdate={onFormHeaderUpdate}
                            onDelete={onFormHeaderDelete}
                        />
                    )}
                    {element.type !== "Header" && <FormInput
                        fieldData={element}
                        index={index}
                        onConfigUpdate={onFormInputConfigUpdate}
                        onDelete={onFormInputDelete}
                    />}
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
            </ElementOuterBox>
        );
    }

    const Toolbar = useMemo(() => {
        const numHeaderItems = headerItems.length;
        const firstHeaderItem = numHeaderItems === 1 ? headerItems[0] : null;
        const DisplayedHeaderIcon = firstHeaderItem ? (() => firstHeaderItem.icon) : HeaderIcon;
        function headerOnClick(event: React.MouseEvent<HTMLElement>) {
            if (firstHeaderItem) {
                handleAddHeader({ tag: firstHeaderItem.tag });
            } else {
                handleHeaderPopoverOpen(event);
            }
        }

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

        return (
            <ToolbarBox formElementsCount={formElements.length}>
                {numHeaderItems > 0 && <>
                    {isMobile ? <IconButton onClick={headerOnClick}>
                        <DisplayedHeaderIcon width={24} height={24} />
                    </IconButton> : <Typography variant="body1" sx={toolbarLargeButtonStyle} onClick={headerOnClick}>
                        {numHeaderItems === 1 ? `Add ${headerItems[0].label.toLowerCase()}` : "Add header"}
                    </Typography>}
                </>}
                {numInputItems > 0 && <>
                    {isMobile ? <IconButton onClick={inputOnClick}>
                        <DisplayedInputIcon width={24} height={24} />
                    </IconButton> : <Typography variant="body1" sx={toolbarLargeButtonStyle} onClick={inputOnClick}>
                        {numInputItems === 1 ? `Add ${inputItems[0].items[0].label.toLowerCase()}` : "Add input"}
                    </Typography>}
                </>}
            </ToolbarBox>
        );
    }, [formElements.length, handleAddHeader, handleAddInput, handleHeaderPopoverOpen, handleInputPopoverOpen, headerItems, inputItems, isMobile]);

    //TODO calculate using defaultValues
    const formInitialValues = useMemo(() => ({}) as Record<string, unknown>, []);

    return (
        <div>
            <Popover
                open={isHeaderPopoverOpen}
                anchorEl={headerAnchorEl}
                onClose={closeHeaderPopover}
                anchorOrigin={popoverAnchorOrigin}
            >
                <List>
                    {headerItems.map((item) => (
                        <HeaderListItem
                            key={item.tag}
                            icon={item.icon}
                            label={item.label}
                            tag={item.tag}
                            onAddHeader={handleAddHeader}
                        />
                    ))}
                </List>
            </Popover>
            <Popover
                open={isInputPopoverOpen}
                anchorEl={inputAnchorEl}
                onClose={closeInputPopover}
                anchorOrigin={popoverAnchorOrigin}
            >

                {inputItems.map(({ category, items }) => (
                    <Fragment key={category}>
                        <ListSubheader>{category}</ListSubheader>
                        {items.map(({ icon, label, type }) => (
                            <InputListItem
                                key={type}
                                icon={icon}
                                label={label}
                                type={type}
                                onAddInput={handleAddInput}
                            />
                        ))}
                        <Divider />
                    </Fragment>
                ))}
            </Popover>
            {formElements.length === 0 && <FormHelperText variant="body1">Use the options below to populate the form.</FormHelperText>}
            {formElements.length === 0 ? Toolbar : null}
            {formElements.length > 0 && <Divider sx={formDividerStyle} />}
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
                                        {formElements.map((element, index) => (
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
