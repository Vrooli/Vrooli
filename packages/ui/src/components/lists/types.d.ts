export interface FeedListProps<DataType> {
    title?: string;
    data: DataType[];
    cardFactory: (node: DataType, index: number) => JSX.Element;
    onClick: (pressedId?: string) => void;
}

export interface ResourceListProps {
    title?: string;
}

export interface StatsListProps {
    data: Array<any>;
}