export interface SlideProps {
    id: string;
    children: React.ReactNode;
    sx?: object;
}

export interface SlideContainerProps {
    id: string;
    children: React.ReactNode;
    sx?: object;
}

export interface SlideContainerNeonProps {
    id: string;
    children: React.ReactNode;
    show?: boolean;
    sx?: object;
}

export interface SlideContentProps {
    children: React.ReactNode;
    id?: string;
    sx?: object;
}

export interface SlidePageProps {
    children: React.ReactNode;
    id?: string;
    sx?: object;
}