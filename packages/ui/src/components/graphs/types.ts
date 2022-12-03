export interface Dimensions {
    width: number | undefined;
    height: number | undefined;
}

export interface BarGraphProps {
    className?: string;
    data?: any;
    dimensions?: Dimensions;
    style?: any;
}