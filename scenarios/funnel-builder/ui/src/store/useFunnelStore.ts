import { create } from 'zustand'
import { Funnel, FunnelStep } from '../types'

interface FunnelStore {
  funnels: Funnel[]
  currentFunnel: Funnel | null
  selectedStep: FunnelStep | null
  isLoading: boolean
  error: string | null
  
  setFunnels: (funnels: Funnel[]) => void
  setCurrentFunnel: (funnel: Funnel | null) => void
  setSelectedStep: (step: FunnelStep | null) => void
  addStep: (step: FunnelStep) => void
  updateStep: (stepId: string, updates: Partial<FunnelStep>) => void
  deleteStep: (stepId: string) => void
  reorderSteps: (startIndex: number, endIndex: number) => void
  updateFunnelSettings: (settings: Partial<Funnel['settings']>) => void
  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void
}

export const useFunnelStore = create<FunnelStore>((set) => ({
  funnels: [],
  currentFunnel: null,
  selectedStep: null,
  isLoading: false,
  error: null,

  setFunnels: (funnels) => set({ funnels }),
  
  setCurrentFunnel: (funnel) => set({ currentFunnel: funnel }),
  
  setSelectedStep: (step) => set({ selectedStep: step }),
  
  addStep: (step) => set((state) => {
    if (!state.currentFunnel) return state
    
    const newFunnel = {
      ...state.currentFunnel,
      steps: [...state.currentFunnel.steps, step]
    }
    
    return {
      currentFunnel: newFunnel,
      selectedStep: step
    }
  }),
  
  updateStep: (stepId, updates) => set((state) => {
    if (!state.currentFunnel) return state
    
    const newSteps = state.currentFunnel.steps.map(step =>
      step.id === stepId ? { ...step, ...updates } : step
    )
    
    const newFunnel = {
      ...state.currentFunnel,
      steps: newSteps
    }
    
    const updatedSelectedStep = state.selectedStep?.id === stepId
      ? { ...state.selectedStep, ...updates }
      : state.selectedStep
    
    return {
      currentFunnel: newFunnel,
      selectedStep: updatedSelectedStep
    }
  }),
  
  deleteStep: (stepId) => set((state) => {
    if (!state.currentFunnel) return state
    
    const newSteps = state.currentFunnel.steps
      .filter(step => step.id !== stepId)
      .map((step, index) => ({ ...step, position: index }))
    
    const newFunnel = {
      ...state.currentFunnel,
      steps: newSteps
    }
    
    return {
      currentFunnel: newFunnel,
      selectedStep: state.selectedStep?.id === stepId ? null : state.selectedStep
    }
  }),
  
  reorderSteps: (startIndex, endIndex) => set((state) => {
    if (!state.currentFunnel) return state
    
    const newSteps = Array.from(state.currentFunnel.steps)
    const [removed] = newSteps.splice(startIndex, 1)
    newSteps.splice(endIndex, 0, removed)
    
    const reorderedSteps = newSteps.map((step, index) => ({
      ...step,
      position: index
    }))
    
    const newFunnel = {
      ...state.currentFunnel,
      steps: reorderedSteps
    }
    
    return { currentFunnel: newFunnel }
  }),
  
  updateFunnelSettings: (settings) => set((state) => {
    if (!state.currentFunnel) return state
    
    const newFunnel = {
      ...state.currentFunnel,
      settings: {
        ...state.currentFunnel.settings,
        ...settings
      }
    }
    
    return { currentFunnel: newFunnel }
  }),
  
  setLoading: (loading) => set({ isLoading: loading }),
  
  setError: (error) => set({ error })
}))