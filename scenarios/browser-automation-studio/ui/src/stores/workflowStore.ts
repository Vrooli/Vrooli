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

interface WorkflowStore {
  workflows: Workflow[];
  currentWorkflow: Workflow | null;
  nodes: Node[];
  edges: Edge[];
  
  loadWorkflows: (projectId?: string) => Promise<void>;
  loadWorkflow: (id: string) => Promise<void>;
  createWorkflow: (name: string, folderPath: string, projectId?: string) => Promise<void>;
  saveWorkflow: () => Promise<void>;
  updateWorkflow: (updates: Partial<Workflow>) => void;
  generateWorkflow: (prompt: string, name: string, folderPath: string, projectId?: string) => Promise<void>;
  modifyWorkflow: (prompt: string) => Promise<void>;
  deleteWorkflow: (id: string) => Promise<void>;
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
        nodes: workflow.nodes || [],
        edges: workflow.edges || []
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
        throw new Error(`Failed to create workflow: ${response.status}`);
      }
      const workflow = await response.json();
      set({ 
        currentWorkflow: workflow,
        nodes: [],
        edges: []
      });
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
        throw new Error(`Failed to generate workflow: ${response.status}`);
      }
      const workflow = await response.json();
      set({ 
        currentWorkflow: workflow,
        nodes: workflow.nodes || [],
        edges: workflow.edges || []
      });
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
        throw new Error(`Failed to modify workflow: ${response.status}`);
      }
      const modifiedWorkflow = await response.json();
      set({ 
        currentWorkflow: modifiedWorkflow,
        nodes: modifiedWorkflow.nodes || [],
        edges: modifiedWorkflow.edges || []
      });
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
  }
}));