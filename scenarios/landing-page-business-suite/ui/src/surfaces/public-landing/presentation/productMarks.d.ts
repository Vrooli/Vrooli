import type { SVGProps } from 'react';
import type { Mark } from './resources';

export const productMarkPaths: Readonly<Record<Mark, string>>;
export const productMarkDrawing: Readonly<Pick<SVGProps<SVGSVGElement>, 'viewBox' | 'fill'> & Pick<SVGProps<SVGPathElement>, 'strokeWidth' | 'strokeLinecap' | 'strokeLinejoin'>>;
export function getProductMarkPath(kind: unknown): string | undefined;
