import Slide from "@mui/material/Slide";
import type { SlideProps } from "@mui/material/Slide";
import { forwardRef } from "react";

export const UpTransition = forwardRef<unknown, SlideProps>(function Transition(props, ref) {
    return <Slide direction="up" ref={ref} {...props} />;
});
UpTransition.displayName = "UpTransition";
