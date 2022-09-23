import { SvgBase } from './base';
import { SvgProps } from './types';

export const Header1Icon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fillOpacity: 1,
                fill: props.fill ?? 'white',
                stroke: 'none',
            }}
            d="m18.34046 5.88114-3.53285 3.0057.50882.69756c.17079.34395.6265.5163.9579.2821.5161-.41032 1.01372-.84552 1.52262-1.26519.19106-.20369.26923-.1871.20044.13507-.021 2.4211-.0021 4.84333-.0077 7.2646h-2.34752v1.48234h6.40848v-1.48233h-2.0835V5.88114Zm-14.34921.00171v11.56798h2.16057v-5.05736h4.71048v5.05736h2.16057V5.88285H10.8623v4.9753H6.15182v-4.9753Z"
            transform="matrix(1.01337 0 0 1.14236 -1.19499 -1.34536)"
        />
    </SvgBase>
)