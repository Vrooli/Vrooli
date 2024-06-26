import { InputType, mergeDeep, noopSubmit } from "@local/shared";
import { Box, Divider, IconButton, List, ListItem, ListItemIcon, ListItemText, ListSubheader, Popover, Theme, Tooltip, Typography, useTheme } from "@mui/material";
import { FormInput } from "components/inputs/form";
import { FormHeader } from "components/inputs/form/FormHeader/FormHeader";
import { Formik } from "formik";
import { FormErrorBoundary } from "forms/FormErrorBoundary/FormErrorBoundary";
import { CreateFormInputProps, createFormInput } from "forms/generators";
import { FormBuildViewProps, FormElement, FormHeaderType, FormInputType } from "forms/types";
import { usePopover } from "hooks/usePopover";
import { useWindowSize } from "hooks/useWindowSize";
import { CaseSensitiveIcon, DragIcon, Header1Icon, Header2Icon, Header3Icon, Header4Icon, HeaderIcon, LinkIcon, ListBulletIcon, ListCheckIcon, ListIcon, NumberIcon, ObjectIcon, SliderIcon, SwitchIcon, CaseSensitiveIcon as TextInputIcon, UploadIcon, VrooliIcon } from "icons";
import { Fragment, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { DragDropContext, Draggable, DraggableProvided, DropResult, Droppable } from "react-beautiful-dnd";
import { randomString } from "utils/codes";

const NUM_HEADERS = 6;
const HEADER_OFFSET = 2;
function getHeaderStyle(tag: FormHeaderType["tag"], typography: Theme["typography"]) {
    // Use style of tag + 2. Ex: h4 -> h6, h5 -> h6
    const tagSize = Math.min(NUM_HEADERS, parseInt(tag[1]) + HEADER_OFFSET);
    return {
        paddingLeft: "8px",
        paddingRight: "8px",
        ...((typography[`h${tagSize}` as keyof typeof typography]) as object || {}),
    };
}


//TODO: 1: Add way to limit what options are available, so we can use for routine type forms (e.g. connect and configure api, code, etc.)
//TODO 2: Add input which selects object from site, so that we can define an input for the routine types that links things like apis.
//TODO 3: Add groups and page breaks
export function FormBuildView({
    display,
    onClose,
}: FormBuildViewProps) {
    const { breakpoints, palette, typography } = useTheme();
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
        setTimeout(() => {
            elementRefs.current[newElements.length - 1]?.scrollIntoView({ behavior: "smooth" });
        }, 100);
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

    // Focus on the selected element
    useEffect(function focusOnSelectedElement() {
        if (selectedElementIndex !== null) {
            elementRefs.current[selectedElementIndex]?.focus();
        }
    }, [selectedElementIndex]);

    const headerItems = [
        { tag: "h1", icon: <Header1Icon />, label: "Title (Largest)" },
        { tag: "h2", icon: <Header2Icon />, label: "Subtitle (Large)" },
        { tag: "h3", icon: <Header3Icon />, label: "Header (Medium)" },
        { tag: "h4", icon: <Header4Icon />, label: "Subheader (Small)" },
    ] as const;

    const inputItems = [
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
    ] as const;

    function handleAddHeader(data: Partial<FormHeaderType>) {
        const tag = data.tag ?? "h1";
        handleAddElement<FormHeaderType>({
            type: "Header",
            label: data.label ?? `New ${tag.toUpperCase()}`,
            tag,
            ...data,
        });
        closeHeaderPopover();
    }

    const handleUpdateHeader = useCallback(function handleUpdateHeaderCallback(index: number, data: Partial<FormHeaderType>) {
        const element = {
            ...formElements[index],
            ...data,
        };
        const newElements = [...formElements.slice(0, index), element, ...formElements.slice(index + 1)] as FormElement[];
        console.log("in handleUpdateHeader", newElements);
        setFormElements(newElements);
    }, [formElements]);

    function handleAddInput(data: Omit<Partial<FormInputType>, "type"> & { type: InputType }) {
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
    }

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
        return (
            <Box
                key={element.id}
                ref={providedDrag.innerRef}
                {...providedDrag.draggableProps}
                sx={{
                    display: "flex",
                    background: palette.background.paper,
                    border: isSelected ? "4px solid" + palette.secondary.main : "none",
                    borderRadius: "8px",
                    overflow: "overlay",
                    height: "100%",
                    alignItems: "stretch",
                }}
            >
                <button
                    ref={ref => elementRefs.current[index] = ref}
                    onClick={() => selectElement(index)}
                    onKeyDown={(e) => {
                        if (e.key === "Enter" || e.key === " ") {
                            e.preventDefault();
                            selectElement(index);
                        }
                    }}
                    style={{
                        padding: 0,
                        width: "100%",
                        overflow: "hidden",
                        background: "none",
                        color: "inherit",
                        border: "none",
                        textAlign: "left",
                        cursor: "pointer",
                    }}
                >
                    {element.type === "Header" && (
                        <FormHeader
                            element={element as FormHeaderType}
                            isSelected={isSelected}
                            onUpdate={(data) => handleUpdateHeader(index, data)}
                            onDelete={() => handleDeleteElement(index)}
                        />
                    )}
                    {element.type !== "Header" && <FormInput
                        fieldData={element}
                        index={index}
                        onConfigUpdate={isSelected ? (updatedInput) => { handleUpdateInput(index, updatedInput); } : undefined}
                        onDelete={isSelected ? () => { handleDeleteElement(index); } : undefined}
                    />}
                    {isSelected ? Toolbar : null}
                </button>
                {isSelected && (
                    <Box
                        {...providedDrag.dragHandleProps}
                        sx={{
                            cursor: "pointer",
                            display: "flex",
                            flexDirection: "column",
                            padding: "4px",
                            background: palette.secondary.main,
                        }}
                    >
                        <DragIcon
                            fill={palette.secondary.contrastText}
                            width="24px"
                            height="24px"
                            style={{ marginTop: "auto", marginBottom: "auto" }}
                        />
                    </Box>
                )}
            </Box>
        );
    }

    const Toolbar = useMemo(() => (
        <Box
            sx={{
                display: "flex",
                justifyContent: "space-around",
                margin: "-4px",
                padding: "4px",
                background: palette.primary.main,
                color: palette.secondary.contrastText,
                borderBottomLeftRadius: formElements.length === 0 ? "8px" : "0px",
                borderBottomRightRadius: formElements.length === 0 ? "8px" : "0px",
            }}
        >
            <Tooltip title="Add Header">
                {isMobile ? <IconButton onClick={handleHeaderPopoverOpen}>
                    <HeaderIcon width={24} height={24} />
                </IconButton> : <Typography variant="body1" sx={{ cursor: "pointer" }} onClick={handleHeaderPopoverOpen}>Header</Typography>}
            </Tooltip>
            <Tooltip title="Add Input">
                {isMobile ? <IconButton onClick={handleInputPopoverOpen}>
                    <TextInputIcon width={24} height={24} />
                </IconButton> : <Typography variant="body1" sx={{ cursor: "pointer" }} onClick={handleInputPopoverOpen}>Input</Typography>}
            </Tooltip>
        </Box>
    ), [formElements.length, handleHeaderPopoverOpen, handleInputPopoverOpen, isMobile, palette.primary.main, palette.secondary.contrastText]);

    return (
        <div>
            <Popover
                open={isHeaderPopoverOpen}
                anchorEl={headerAnchorEl}
                onClose={closeHeaderPopover}
                anchorOrigin={{
                    vertical: "bottom",
                    horizontal: "center",
                }}
            >
                <List>
                    {headerItems.map(({ icon, label, tag }) => (
                        <ListItem
                            key={tag}
                            button
                            onClick={function addHeaderClick() { handleAddHeader({ tag }); }}
                        >
                            <ListItemIcon>{icon}</ListItemIcon>
                            <ListItemText primary={label} />
                        </ListItem>
                    ))}
                </List>
            </Popover>
            <Popover
                open={isInputPopoverOpen}
                anchorEl={inputAnchorEl}
                onClose={closeInputPopover}
                anchorOrigin={{
                    vertical: "bottom",
                    horizontal: "center",
                }}
            >

                {inputItems.map(({ category, items }) => (
                    <Fragment key={category}>
                        <ListSubheader>{category}</ListSubheader>
                        {items.map(({ icon, label, type }) => (
                            <ListItem
                                key={type}
                                button
                                onClick={() => handleAddInput({ type })}
                            >
                                <ListItemIcon>{icon}</ListItemIcon>
                                <ListItemText primary={label} />
                            </ListItem>
                        ))}
                        <Divider />
                    </Fragment>
                ))}
            </Popover>
            {formElements.length === 0 && <Typography variant="body1" sx={{ textAlign: "center", color: palette.text.secondary, padding: "20px" }}>Use the options below to populate the form.</Typography>}
            {formElements.length === 0 ? Toolbar : null}
            {formElements.length > 0 && <Divider sx={{ marginBottom: 2 }} />}
            {/* Formik to handle form state. Even though we're not entering data, we need to make sure we aren't using a formik context higher up in the tree */}
            <Formik
                enableReinitialize={true}
                initialValues={{}} //TODO calculate using defaultValues
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
