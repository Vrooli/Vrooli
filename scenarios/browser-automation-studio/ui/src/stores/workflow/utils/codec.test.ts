import { describe, expect, it } from 'vitest';
import { canvasToWorkflowDefinition, workflowDefinitionToCanvas, workflowDefinitionToProto } from './codec';
import type { Node, Edge } from 'reactflow';

describe('workflow codec behavior', () => {
  it('round-trips V2 action identity, positions, handles, metadata, settings, and forward fields', () => {
    const source = {
      metadata: { name: 'Checkout', description: 'test', labels: { team: 'qa' }, version: '2' },
      settings: { viewport_width: 1440, viewport_height: 900, futureSetting: { mode: 'keep' } },
      futureDefinitionField: { enabled: true },
      nodes: [{ id: 'n1', action: { type: 'ACTION_TYPE_CLICK', click: { selector: '#buy' } }, position: { x: 90, y: 120 }, execution_settings: { timeout_ms: 2500 }, futureNodeField: 'keep' }],
      edges: [{ id: 'e1', source: 'n1', target: 'n1', source_handle: 'out', target_handle: 'in', futureEdgeField: 7 }],
    };
    const canvas = workflowDefinitionToCanvas(source);
    const result = canvasToWorkflowDefinition(source, canvas.nodes, canvas.edges);

    expect(result.nodes).toEqual(source.nodes);
    expect(result.edges).toEqual(source.edges);
    expect(result.metadata).toEqual(source.metadata);
    expect(result.settings).toEqual(source.settings);
    expect(result.futureDefinitionField).toEqual(source.futureDefinitionField);
    const proto = workflowDefinitionToProto(result);
    expect(proto.nodes[0].action?.type).toBeDefined();
    expect(proto.nodes[0].executionSettings?.timeoutMs).toBe(2500);
    expect(proto.settings?.viewportWidth).toBe(1440);
  });

  it('keeps legacy import routing explicit and creates stable canvas defaults', () => {
    const result = workflowDefinitionToCanvas({ nodes: [{ id: 'old', type: 'navigate', data: { url: '/' } }], edges: [] });
    expect(result.nodes[0]).toMatchObject({ id: 'old', type: 'navigate', position: { x: 100, y: 100 } });
    expect(result.nodes[0].action).toBeUndefined();
  });

  it('strips React Flow transient and preview state while keeping viewport settings', () => {
    const nodes = [{
      id: 'n1', type: 'click', position: { x: 1, y: 2 }, selected: true,
      data: { selector: '#buy', previewScreenshot: 'image' },
      action: { type: 'ACTION_TYPE_CLICK', click: { selector: '#buy' } },
    }] as unknown as Node[];
    const edges = [{ id: 'e1', source: 'n1', target: 'n1', sourceHandle: 'out', selected: true }] as Edge[];
    const result = canvasToWorkflowDefinition({ futureDefinitionField: true }, nodes, edges, { width: 1440, height: 900 });
    expect(result.nodes[0]).toMatchObject({ id: 'n1', position: { x: 1, y: 2 } });
    expect(result.nodes[0]).not.toHaveProperty('selected');
    expect(result.nodes[0]).not.toHaveProperty('data');
    expect(result.edges[0]).toMatchObject({ id: 'e1', source_handle: 'out' });
    expect(result.edges[0]).not.toHaveProperty('selected');
    expect(result.settings).toEqual({ executionViewport: { width: 1440, height: 900 }, viewport_width: 1440, viewport_height: 900 });
    expect(result.futureDefinitionField).toBe(true);
    expect(workflowDefinitionToProto(result).settings?.viewportHeight).toBe(900);
  });

  it('rejects invalid executable nodes before persistence', () => {
    expect(() => canvasToWorkflowDefinition({}, [{ id: 'n1', position: { x: 0, y: 0 } } as Node], [])).toThrow('missing a V2 action');
  });
});
