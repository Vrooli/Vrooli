import { Box, IconButton, Modal, ModalProps, styled } from "@mui/material";
import { CloseIcon } from "icons";
import { useCallback } from "react";

interface ImageModalProps extends Omit<ModalProps, "zIndex"> {
    zIndex?: number;
}
const ImageModal = styled(Modal, {
    shouldForwardProp: (prop) => prop !== "zIndex",
})<ImageModalProps>(({ zIndex }) => ({
    backgroundColor: "rgba(0,0,0,0.8)",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    zIndex,
}));
const ImageModalInner = styled(Box)(() => ({
    position: "relative",
    width: "80%",
    height: "80%",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
}));
const ImageButton = styled("button")(() => ({
    background: "none",
    border: "none",
    padding: 0,
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    cursor: "pointer",
    maxWidth: "100%",
    maxHeight: "100%",
}));
const StyledImg = styled("img")(() => ({
    maxWidth: "100%",
    maxHeight: "100%",
    objectFit: "contain",
}));
const CloseIconButton = styled(IconButton)(() => ({
    position: "absolute",
    right: 20,
    top: 20,
    color: "white",
}));

type ImagePopupProps = {
    alt: string;
    onClose: () => unknown;
    open: boolean;
    src: string;
    zIndex?: number;
};

export function ImagePopup({
    alt,
    onClose,
    open,
    src,
    zIndex,
}: ImagePopupProps) {
    const stopPropagation = useCallback((e: React.MouseEvent) => {
        e.stopPropagation();
    }, []);

    return (
        <ImageModal
            aria-describedby="full-size-screenshot"
            aria-labelledby="full-size-image"
            onClose={onClose}
            open={open}
            zIndex={zIndex}
        >
            <ImageModalInner onClick={onClose}>
                <ImageButton
                    onClick={stopPropagation}
                    aria-label={alt}
                >
                    <StyledImg
                        src={src}
                        alt={alt}
                    />
                </ImageButton>
                <CloseIconButton
                    aria-label="close"
                    onClick={onClose}
                >
                    <CloseIcon />
                </CloseIconButton>
            </ImageModalInner>
        </ImageModal>
    );
}
