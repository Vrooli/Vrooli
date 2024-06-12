import { Box, Divider, IconButton, List, ListItem, ListItemIcon, ListItemText, Popover, Tooltip, Typography, useTheme } from "@mui/material";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { FormBuildViewProps } from "forms/types";
import { usePopover } from "hooks/usePopover";
import { DeleteIcon, EditIcon, Header1Icon, Header2Icon, Header3Icon, Header4Icon, Header5Icon, Header6Icon, HeaderIcon, CaseSensitiveIcon as TextInputIcon } from "icons";
import { createElement, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

type BaseElement = {
    label: string;
    id: string;
};
type HeaderElement = BaseElement & {
    type: "Header";
    tag: "h1" | "h2" | "h3" | "h4" | "h5" | "h6";
};
type TextInputElement = BaseElement & {
    type: "TextInput";
    placeholder: string;
};
type FormElement = HeaderElement | TextInputElement;

//TODO: 1: Add way to limit what options are available, so we can use for routine type forms (e.g. connect and configure api, code, etc.)
//TODO 2: Add input which selects object from site, so that we can define an input for the routine types that links things like apis.
//TODO 3: Changing the name, description, and helper text of an input should be hidden away until the edit button of the selected input is pressed
export const FormBuildView = ({
    display,
    onClose,
}: FormBuildViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [formElements, setFormElements] = useState<FormElement[]>([]);
    const [selectedElementIndex, setSelectedElementIndex] = useState<number | null>(null);
    const elementRefs = useRef<(HTMLDivElement | null)[]>([]);

    const [anchorEl, openPopover, closePopover, isPopoverOpen] = usePopover();
    const handlePopoverOpen = useCallback((event: React.MouseEvent<HTMLElement>) => {
        event.preventDefault();
        openPopover(event);
    }, [openPopover]);

    const handleAddElement = useCallback(<T extends FormElement>(element: Omit<T, "id">) => {
        const newElement = { ...element, id: Date.now().toString() } as unknown as FormElement;
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

    const handleDeleteElement = (index: number) => {
        const newElements = [...formElements];
        newElements.splice(index, 1);
        setFormElements(newElements);
        setSelectedElementIndex(null); // Reset selection
    };

    const selectElement = (index: number) => {
        setSelectedElementIndex(index);
        elementRefs.current[index]?.scrollIntoView({ behavior: "smooth" });
        // // Focus on the selected element
        // setTimeout(() => {
        //     elementRefs.current[index]?.focus();
        // }, 100);
    };

    // Focus on the selected element
    useEffect(() => {
        if (selectedElementIndex !== null) {
            elementRefs.current[selectedElementIndex]?.focus();
        }
    }, [selectedElementIndex]);

    const headerItems = [
        { tag: "h1", icon: <Header1Icon />, label: t("Header1") },
        { tag: "h2", icon: <Header2Icon />, label: t("Header2") },
        { tag: "h3", icon: <Header3Icon />, label: t("Header3") },
        { tag: "h4", icon: <Header4Icon />, label: t("Header4") },
        { tag: "h5", icon: <Header5Icon />, label: t("Header5") },
        { tag: "h6", icon: <Header6Icon />, label: t("Header6") },
    ] as const;

    const handleAddHeader = (tag: HeaderElement["tag"]) => {
        handleAddElement<HeaderElement>({ type: "Header", label: `New ${tag.toUpperCase()}`, tag });
        closePopover();
    };

    const handleUpdateHeader = useCallback((index: number, label: string) => {
        const newElements = [...formElements];
        const element = newElements[index] as HeaderElement;
        element.label = label;
        setFormElements(newElements);
    }, [formElements]);

    const renderElement = (element: FormElement, index: number) => {
        const isSelected = selectedElementIndex === index;
        return (
            <Box key={element.id} sx={{ display: "flex", alignItems: "center", margin: "5px" }}>
                <div
                    ref={ref => elementRefs.current[index] = ref}
                    onClick={() => selectElement(index)}
                    style={{
                        border: isSelected ? "4px solid" + palette.secondary.main : "none",
                        borderRadius: "8px",
                        margin: "5px",
                        overflow: "overlay",
                        width: "100%",
                    }}
                >
                    {element.type === "Header"
                        ? isSelected
                            ? <TextInput
                                fullWidth
                                variant="outlined"
                                defaultValue={element.label}
                                onBlur={(e) => handleUpdateHeader(index, e.target.value)}
                                autoFocus
                            />
                            : createElement(element.tag, { style: { paddingLeft: "8px", paddingRight: "8px" } }, element.label) : element.label
                    }
                    {isSelected ? Toolbar : null}
                </div>
                {isSelected && (
                    <Box sx={{ display: "flex", flexDirection: "column" }}>
                        <Tooltip title={t("Edit")}>
                            <IconButton onClick={() => console.log("Edit element", index)}>
                                <EditIcon fill={palette.secondary.main} />
                            </IconButton>
                        </Tooltip>
                        <Tooltip title={t("Delete")}>
                            <IconButton onClick={() => handleDeleteElement(index)}>
                                <DeleteIcon fill={palette.error.main} />
                            </IconButton>
                        </Tooltip>
                    </Box>
                )}
            </Box>
        );
    };

    const Toolbar = useMemo(() => (
        <Box
            sx={{
                display: "flex",
                justifyContent: "space-around",
                padding: "4px",
                background: palette.primary.main,
                color: palette.secondary.contrastText,
                borderBottomLeftRadius: formElements.length === 0 ? "8px" : "0px",
                borderBottomRightRadius: formElements.length === 0 ? "8px" : "0px",
            }}
        >
            <Tooltip title="Add Header">
                <IconButton onClick={handlePopoverOpen}>
                    <HeaderIcon width={24} height={24} />
                </IconButton>
            </Tooltip>
            <Tooltip title="Add Text Input">
                <IconButton onClick={() => handleAddElement<TextInputElement>({ type: "TextInput", label: "New Text Input", placeholder: "Placeholder" })}>
                    <TextInputIcon width={24} height={24} />
                </IconButton>
            </Tooltip>
        </Box>
    ), [formElements.length, handleAddElement, handlePopoverOpen, palette.primary.main, palette.secondary.contrastText]);

    return (
        <div>
            <Popover
                open={isPopoverOpen}
                anchorEl={anchorEl}
                onClose={closePopover}
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
                            onClick={() => handleAddHeader(tag)}
                        >
                            <ListItemIcon>{icon}</ListItemIcon>
                            <ListItemText primary={label} />
                        </ListItem>
                    ))}
                </List>
            </Popover>
            {formElements.length === 0 && <Typography variant="body1" sx={{ textAlign: "center", color: palette.text.secondary, padding: "20px" }}>Use the options below to populate the form.</Typography>}
            {formElements.length === 0 ? Toolbar : null}
            {formElements.length > 0 && <Divider />}
            {formElements.map(renderElement)}
        </div>
    );
};
