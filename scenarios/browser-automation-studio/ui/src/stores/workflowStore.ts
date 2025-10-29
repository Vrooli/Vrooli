import { create } from 'zustand';
import { Node, Edge } from 'reactflow';
import { getConfig } from '../config';

interface Workflow {
  id: string;
  projectId?: string;
  name: string;
  folderPath: string;
  nodes: Node[];
  edges: Edge[];
  createdAt: Date;
  updatedAt: Date;
}

const normalizeNodes = (nodes: any[] | undefined | null): Node[] => {
  if (!Array.isArray(nodes)) return [];
  return nodes.map((node, index) => {
    const id = node?.id ? String(node.id) : `node-${index + 1}`;
    const type = node?.type ? String(node.type) : 'navigate';
    const position = {
      x: Number(node?.position?.x ?? 100 + index * 200) || 0,
      y: Number(node?.position?.y ?? 100 + index * 120) || 0,
    };
    const data = node?.data && typeof node.data === 'object' ? node.data : {};
    return {
      ...node,
      id,
      type,
      position,
      data,
    } as Node;
  });
};

const normalizeEdges = (edges: any[] | undefined | null): Edge[] => {
  if (!Array.isArray(edges)) return [];
  return edges
    .map((edge, index) => {
      const id = edge?.id ? String(edge.id) : `edge-${index + 1}`;
      const source = edge?.source ? String(edge.source) : '';
      const target = edge?.target ? String(edge.target) : '';
      if (!source || !target) return null;
      return {
        ...edge,
        id,
        source,
        target,
      } as Edge;
    })
    .filter(Boolean) as Edge[];
};

interface WorkflowStore {
  workflows: Workflow[];
  currentWorkflow: Workflow | null;
  nodes: Node[];
  edges: Edge[];
  
  loadWorkflows: (projectId?: string) => Promise<void>;
  loadWorkflow: (id: string) => Promise<void>;
  createWorkflow: (name: string, folderPath: string, projectId?: string) => Promise<Workflow>;
  saveWorkflow: () => Promise<void>;
  updateWorkflow: (updates: Partial<Workflow>) => void;
  generateWorkflow: (prompt: string, name: string, folderPath: string, projectId?: string) => Promise<Workflow>;
  modifyWorkflow: (prompt: string) => Promise<Workflow>;
  deleteWorkflow: (id: string) => Promise<void>;
  bulkDeleteWorkflows: (projectId: string, workflowIds: string[]) => Promise<string[]>;
}

export const useWorkflowStore = create<WorkflowStore>((set, get) => ({
  workflows: [],
  currentWorkflow: null,
  nodes: [],
  edges: [],

  normalizeNodes: (nodes: any[] | undefined | null) => {
    if (!Array.isArray(nodes)) return [];
    return nodes
      .map((node, index) => {
        const id = node?.id ? String(node.id) : `node-${index + 1}`;
        const type = node?.type ? String(node.type) : 'navigate';
        const position = {
          x: Number(node?.position?.x ?? 100 + index * 200) || 0,
          y: Number(node?.position?.y ?? 100 + index * 120) || 0,
        };
        const data = node?.data && typeof node.data === 'object' ? node.data : {};
        return {
          ...node,
          id,
          type,
          position,
          data,
        } as Node;
      });
  },

  normalizeEdges: (edges: any[] | undefined | null) => {
    if (!Array.isArray(edges)) return [];
    return edges
      .map((edge, index) => {
        const id = edge?.id ? String(edge.id) : `edge-${index + 1}`;
        const source = edge?.source ? String(edge.source) : '';
        const target = edge?.target ? String(edge.target) : '';
        if (!source || !target) return null;
        return {
          ...edge,
          id,
          source,
          target,
        } as Edge;
      })
      .filter(Boolean) as Edge[];
  },
  
  loadWorkflows: async (projectId?: string) => {
    try {
      const config = await getConfig();
      let url = `${config.API_URL}/workflows`;
      if (projectId) {
        url = `${config.API_URL}/projects/${projectId}/workflows`;
      }
      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(`Failed to load workflows: ${response.status}`);
      }
      const data = await response.json();
      set({ workflows: data.workflows });
    } catch (error) {
      console.error('Failed to load workflows:', error);
    }
  },
  
  loadWorkflow: async (id: string) => {
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/workflows/${id}`);
      if (!response.ok) {
        throw new Error(`Failed to load workflow: ${response.status}`);
      }
      const workflow = await response.json();
      set({ 
        currentWorkflow: workflow,
        nodes: normalizeNodes(workflow.nodes),
        edges: normalizeEdges(workflow.edges)
      });
    } catch (error) {
      console.error('Failed to load workflow:', error);
    }
  },
  
  createWorkflow: async (name: string, folderPath: string, projectId?: string) => {
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/workflows/create`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          project_id: projectId,
          name,
          folder_path: folderPath,
          flow_definition: {
            nodes: [],
            edges: []
          }
        }),
      });
      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to create workflow: ${response.status}`);
      }
      const workflow = await response.json();
      set({ 
        currentWorkflow: workflow,
        nodes: [],
        edges: []
      });
      return workflow;
    } catch (error) {
      console.error('Failed to create workflow:', error);
      throw error;
    }
  },
  
  saveWorkflow: async () => {
    const { currentWorkflow, nodes, edges } = get();
    if (!currentWorkflow) return;
    
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/workflows/${currentWorkflow.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          ...currentWorkflow,
          nodes,
          edges,
          flow_definition: { nodes, edges }
        }),
      });
      if (!response.ok) {
        throw new Error(`Failed to save workflow: ${response.status}`);
      }
      set({ 
        currentWorkflow: { 
          ...currentWorkflow, 
          nodes, 
          edges,
          updatedAt: new Date()
        }
      });
    } catch (error) {
      console.error('Failed to save workflow:', error);
      throw error;
    }
  },
  
  updateWorkflow: (updates: Partial<Workflow>) => {
    const { currentWorkflow } = get();
    if (!currentWorkflow) return;
    
    const updatedWorkflow = { ...currentWorkflow, ...updates };
    set({ 
      currentWorkflow: updatedWorkflow,
      nodes: updates.nodes || get().nodes,
      edges: updates.edges || get().edges
    });
  },
  
  generateWorkflow: async (prompt: string, name: string, folderPath: string, projectId?: string) => {
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/workflows/create`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          project_id: projectId,
          name,
          folder_path: folderPath,
          ai_prompt: prompt
        }),
      });
      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to generate workflow: ${response.status}`);
      }
      const workflow = await response.json();
      set({ 
        currentWorkflow: workflow,
        nodes: normalizeNodes(workflow.nodes),
        edges: normalizeEdges(workflow.edges)
      });
      return workflow;
    } catch (error) {
      console.error('Failed to generate workflow:', error);
      throw error;
    }
  },
  
  modifyWorkflow: async (prompt: string) => {
    const { currentWorkflow, nodes, edges } = get();
    if (!currentWorkflow) {
      throw new Error('No workflow loaded to modify');
    }
    
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/workflows/${currentWorkflow.id}/modify`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          modification_prompt: prompt,
          current_flow: { nodes, edges }
        }),
      });
      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to modify workflow: ${response.status}`);
      }
      const modifiedWorkflow = await response.json();
      set({ 
        currentWorkflow: modifiedWorkflow,
        nodes: normalizeNodes(modifiedWorkflow.nodes),
        edges: normalizeEdges(modifiedWorkflow.edges)
      });
      return modifiedWorkflow;
    } catch (error) {
      console.error('Failed to modify workflow:', error);
      throw error;
    }
  },
  
  deleteWorkflow: async (id: string) => {
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/workflows/${id}`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        throw new Error(`Failed to delete workflow: ${response.status}`);
      }
      const workflows = get().workflows.filter(w => w.id !== id);
      set({ workflows });
      if (get().currentWorkflow?.id === id) {
        set({ currentWorkflow: null, nodes: [], edges: [] });
      }
    } catch (error) {
      console.error('Failed to delete workflow:', error);
      throw error;
    }
  },

  bulkDeleteWorkflows: async (projectId: string, workflowIds: string[]) => {
    if (workflowIds.length === 0) {
      return [];
    }

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/projects/${projectId}/workflows/bulk-delete`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ workflow_ids: workflowIds }),
      });

      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to delete workflows: ${response.status}`);
      }

      const data = await response.json();
      const deletedIds = Array.isArray(data.deleted_ids) ? (data.deleted_ids as string[]) : workflowIds;
      const deletedSet = new Set(deletedIds);

      set((state) => {
        const currentDeleted = state.currentWorkflow && deletedSet.has(state.currentWorkflow.id);
        return {
          workflows: state.workflows.filter((workflow) => !deletedSet.has(workflow.id)),
          currentWorkflow: currentDeleted ? null : state.currentWorkflow,
          nodes: currentDeleted ? [] : state.nodes,
          edges: currentDeleted ? [] : state.edges,
        };
      });

      return deletedIds;
    } catch (error) {
      console.error('Failed to bulk delete workflows:', error);
      throw error;
    }
  }
}));
