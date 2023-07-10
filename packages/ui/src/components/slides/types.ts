import { ReactNode } from "react";

export interface SlideProps {
    id: string;
    children: ReactNode;
    sx?: object;
}

export interface SlideContainerNeonProps {
    id: string;
    children: ReactNode;
    show?: boolean;
    sx?: object;
}
