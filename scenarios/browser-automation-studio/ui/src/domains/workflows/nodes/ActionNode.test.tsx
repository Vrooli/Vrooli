import { fireEvent, render, screen } from '@/test-utils';
import ActionNode from './ActionNode';
import { nodeTypes } from '../builder/nodeTypes';
import { ActionDefinitionSchema } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';

const harness = vi.hoisted(() => ({
  type: 'click',
  actionType: 'ACTION_TYPE_CLICK' as const,
  params: { selector: '#submit' } as Record<string, unknown>,
  data: {} as Record<string, unknown>,
  updateParams: vi.fn(),
  updateData: vi.fn(),
}));

vi.mock('reactflow', async (importOriginal) => {
  const actual = await importOriginal<typeof import('reactflow')>();
  return {
    ...actual,
    Handle: ({ id, type }: { id?: string; type: string }) => <span data-testid={`handle-${id ?? type}`} />,
    useReactFlow: () => ({ getNodes: () => [{ id: 'node-1', type: harness.type, data: harness.data, action: { type: harness.actionType } }] }),
  };
});
vi.mock('@hooks/useActionParams', () => ({
  useActionParams: () => ({ params: harness.params, updateParams: harness.updateParams }),
}));
vi.mock('@hooks/useNodeData', () => ({
  useNodeData: () => ({
    getValue: (key: string) => harness.data[key],
    updateData: harness.updateData,
  }),
}));
vi.mock('@hooks/useElementPicker', () => ({ useElementPicker: () => ({ onSelect: vi.fn() }) }));
vi.mock('@hooks/useUrlInheritance', () => ({ useUrlInheritance: () => ({ effectiveUrl: 'https://example.test' }) }));

describe('ActionNode', () => {
  beforeEach(() => {
    harness.type = 'click';
    harness.actionType = 'ACTION_TYPE_CLICK';
    harness.params = { selector: '#submit' };
    harness.data = {};
    harness.updateParams.mockClear();
    harness.updateData.mockClear();
  });

  it('edits V2 action params through the shared selector control', () => {
    render(<ActionNode id="node-1" selected={false} data={{}} type="click" />);

    expect(screen.getByText('Click')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('CSS selector...')).toHaveValue('#submit');
    fireEvent.change(screen.getByPlaceholderText('CSS selector...'), { target: { value: '#save' } });
    expect(harness.updateParams).toHaveBeenCalledWith({ selector: '#save' });
  });

  it('routes every supported canvas node type through the shared renderer', () => {
    const allV2ActionTypes = ActionDefinitionSchema.fields
      .filter((field) => field.oneof?.name === 'params' && field.fieldKind === 'message')
      .map((field) => field.localName);
    expect(Object.keys(nodeTypes)).toEqual(expect.arrayContaining(allV2ActionTypes));
    expect(Object.values(nodeTypes).every((renderer) => renderer === ActionNode)).toBe(true);
  });

  it('shows the selected conditional fields and keeps branch handle identifiers', () => {
    harness.type = 'conditional';
    harness.actionType = 'ACTION_TYPE_CONDITIONAL';
    harness.params = { conditionType: 'element', selector: '#ready' };
    render(<ActionNode id="node-1" selected={false} data={{}} type="conditional" />);

    expect(screen.getByText('Conditional')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('CSS selector...')).toBeInTheDocument();
    expect(screen.queryByPlaceholderText('return true;')).not.toBeInTheDocument();
    expect(screen.getByTestId('handle-ifTrue')).toBeInTheDocument();
    expect(screen.getByTestId('handle-ifFalse')).toBeInTheDocument();
    expect(screen.queryByTestId('handle-source')).not.toBeInTheDocument();
  });

  it('keeps loop control handles for existing loop edges', () => {
    harness.type = 'loop';
    harness.actionType = 'ACTION_TYPE_LOOP';
    harness.params = { loopType: 'repeat', count: 3 };
    render(<ActionNode id="node-1" selected={false} data={{}} type="loop" />);

    expect(screen.getByTestId('handle-loopBody')).toBeInTheDocument();
    expect(screen.getByTestId('handle-loopAfter')).toBeInTheDocument();
    expect(screen.getByTestId('handle-loopContinue')).toBeInTheDocument();
    expect(screen.getByTestId('handle-loopBreak')).toBeInTheDocument();
    expect(screen.getByDisplayValue('3')).toBeInTheDocument();
    expect(screen.queryByLabelText('Array variable')).not.toBeInTheDocument();
  });

  it('replaces competing V2 oneof selection fields when the chosen value changes', () => {
    harness.type = 'select';
    harness.actionType = 'ACTION_TYPE_SELECT';
    harness.params = { selector: 'select[name=region]', value: 'north', label: 'North' };
    harness.data = { selectBy: 'value' };
    render(<ActionNode id="node-1" selected={false} data={{}} type="select" />);

    fireEvent.change(screen.getByDisplayValue('north'), { target: { value: 'south' } });
    expect(harness.updateParams).toHaveBeenCalledWith({ label: undefined, index: undefined, value: 'south' });
  });
});
