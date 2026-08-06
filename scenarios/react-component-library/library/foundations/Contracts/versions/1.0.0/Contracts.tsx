/** @vrooliComponentSource foundations.contracts */
export type Density = 'comfortable' | 'compact';
export type Direction = 'ltr' | 'rtl';
export type LayerName = 'base' | 'raised' | 'popover' | 'modal' | 'toast';
export type AsyncStatus = 'idle' | 'pending' | 'success' | 'error' | 'aborted';
export type ControllableValue<T> = { value?: T; defaultValue?: T; onChange?: (value: T) => void };
export type FocusReturnTarget = HTMLElement | null | (() => HTMLElement | null);
export interface DismissReason { source: 'escape' | 'outside' | 'programmatic'; originalEvent?: Event }
export interface StateTransition<T> { status: AsyncStatus; value?: T; error?: unknown }
