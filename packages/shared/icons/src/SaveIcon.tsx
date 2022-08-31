import { SvgProps } from './types';

export const SaveIcon = (props: SvgProps) => (
    <svg
        id={props.id}
        style={props.style}
        width={props.width ?? '24px'}
        height={props.height ?? '24px'}
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg">
        <path
            style={{
                fill: props.fill ?? 'white',
                fillOpacity: 0,
                stroke: props.fill ?? 'white',
                strokeWidth: "1.89",
                strokeLinecap: "round",
            }}
            d="m7.74103 12.51303 2.663 2.64358 5.85494-5.81226"
        />
        <path
            style={{
                fill: props.fill ?? 'white',
                fillOpacity: 0,
                stroke: props.fill ?? 'white',
                strokeWidth: "1.89",
                strokeLinecap: "round",
            }}
            d="m12.74947 3.53197-.40954.36704-.40954.36705V2.79787l.40954.36705zm-1.81144-.29A8.82306 8.82306 0 0 0 3.17773 12a8.82306 8.82306 0 0 0 2.675 6.32753m6.09414 1.93785.40954.36705.40954.36705v-1.4682l-.40954.36705zm1.92977.35483A8.82306 8.82306 0 0 0 20.82227 12a8.82306 8.82306 0 0 0-2.32138-5.96364"
            fill="none"
        />
    </svg>
)