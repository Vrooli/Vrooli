import { ReactNode } from "react";
import { SxType } from "types";

export interface SlideProps {
    id: string;
    children: ReactNode;
    sx?: SxType;
}

export interface SlideContainerNeonProps {
    id: string;
    children: ReactNode;
    show?: boolean;
    sx?: SxType;
}
