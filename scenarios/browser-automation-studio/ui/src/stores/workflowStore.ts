import { create } from 'zustand';
import { Node, Edge } from 'reactflow';
import { getConfig } from '../config';
import { logger } from '../utils/logger';
import { normalizeNodes, normalizeEdges } from '../utils/workflowNormalizers';

export interface Workflow {
  id: string;
  projectId?: string;
  name: string;
  description?: string;
  folderPath: string;
  nodes: Node[];
  edges: Edge[];
  createdAt: Date;
  updatedAt: Date;
  // Index signature for API compatibility
  [key: string]: unknown;
}

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
      logger.error('Failed to load workflows', { component: 'WorkflowStore', action: 'loadWorkflows', projectId }, error);
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
      logger.error('Failed to load workflow', { component: 'WorkflowStore', action: 'loadWorkflow', workflowId: id }, error);
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
      logger.error('Failed to create workflow', { component: 'WorkflowStore', action: 'createWorkflow', name, projectId }, error);
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
      logger.error('Failed to save workflow', { component: 'WorkflowStore', action: 'saveWorkflow', workflowId: currentWorkflow.id }, error);
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
      logger.error('Failed to generate workflow', { component: 'WorkflowStore', action: 'generateWorkflow', name, projectId }, error);
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
      logger.error('Failed to modify workflow', { component: 'WorkflowStore', action: 'modifyWorkflow', workflowId: currentWorkflow.id }, error);
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
      logger.error('Failed to delete workflow', { component: 'WorkflowStore', action: 'deleteWorkflow', workflowId: id }, error);
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
      logger.error('Failed to bulk delete workflows', { component: 'WorkflowStore', action: 'bulkDeleteWorkflows', projectId, count: workflowIds.length }, error);
      throw error;
    }
  }
}));
