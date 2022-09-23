import { SvgBase } from './base';
import { SvgProps } from './types';

export const ItalicIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fillOpacity: 1,
                fill: props.fill ?? 'white',
                stroke: 'none',
            }}
            d="M8.29063 17.45031H2.37734l.47656-2.0625h1.89844L6.48672 7.88H4.58828l.47656-2.0625h5.91329L10.50156 7.88H8.60313l-1.73438 7.50781H8.7672z"
            transform="translate(2.68957 -4.22058) scale(1.39425)"
        />
    </SvgBase>
)