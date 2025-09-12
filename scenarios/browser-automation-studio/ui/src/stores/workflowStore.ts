import { create } from 'zustand';
import { Node, Edge } from 'reactflow';
import axios from 'axios';
import { API_BASE } from '../config';

interface Workflow {
  id: string;
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
  
  loadWorkflows: () => Promise<void>;
  loadWorkflow: (id: string) => Promise<void>;
  createWorkflow: (name: string, folderPath: string) => Promise<void>;
  saveWorkflow: () => Promise<void>;
  updateWorkflow: (updates: Partial<Workflow>) => void;
  generateWorkflow: (prompt: string, name: string, folderPath: string) => Promise<void>;
  deleteWorkflow: (id: string) => Promise<void>;
}

export const useWorkflowStore = create<WorkflowStore>((set, get) => ({
  workflows: [],
  currentWorkflow: null,
  nodes: [],
  edges: [],
  
  loadWorkflows: async () => {
    try {
      const response = await axios.get(`${API_BASE}/workflows`);
      set({ workflows: response.data.workflows });
    } catch (error) {
      console.error('Failed to load workflows:', error);
    }
  },
  
  loadWorkflow: async (id: string) => {
    try {
      const response = await axios.get(`${API_BASE}/workflows/${id}`);
      const workflow = response.data;
      set({ 
        currentWorkflow: workflow,
        nodes: workflow.nodes || [],
        edges: workflow.edges || []
      });
    } catch (error) {
      console.error('Failed to load workflow:', error);
    }
  },
  
  createWorkflow: async (name: string, folderPath: string) => {
    try {
      const response = await axios.post(`${API_BASE}/workflows/create`, {
        name,
        folder_path: folderPath,
        flow_definition: {
          nodes: [],
          edges: []
        }
      });
      const workflow = response.data;
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
      await axios.put(`${API_BASE}/workflows/${currentWorkflow.id}`, {
        ...currentWorkflow,
        nodes,
        edges,
        flow_definition: { nodes, edges }
      });
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
  
  generateWorkflow: async (prompt: string, name: string, folderPath: string) => {
    try {
      const response = await axios.post(`${API_BASE}/workflows/create`, {
        name,
        folder_path: folderPath,
        ai_prompt: prompt
      });
      const workflow = response.data;
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
  
  deleteWorkflow: async (id: string) => {
    try {
      await axios.delete(`${API_BASE}/workflows/${id}`);
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