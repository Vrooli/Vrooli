export type StepType = 'quiz' | 'form' | 'content' | 'cta'

export interface FunnelStep {
  id: string
  type: StepType
  position: number
  title: string
  content: StepContent
  branchingRules?: BranchingRule[]
}

export interface QuizContent {
  question: string
  options: QuizOption[]
  multiSelect?: boolean
}

export interface QuizOption {
  id: string
  text: string
  icon?: string
  image?: string
  nextStepId?: string
}

export interface FormContent {
  fields: FormField[]
  submitText: string
}

export interface FormField {
  id: string
  type: 'text' | 'email' | 'tel' | 'number' | 'textarea' | 'select' | 'checkbox'
  label: string
  placeholder?: string
  required?: boolean
  options?: string[]
  validation?: FieldValidation
}

export interface FieldValidation {
  pattern?: string
  minLength?: number
  maxLength?: number
  min?: number
  max?: number
  message?: string
}

export interface ContentStep {
  headline: string
  body: string
  media?: {
    type: 'image' | 'video'
    url: string
    alt?: string
  }
  buttonText: string
}

export interface CTAContent {
  headline: string
  subheadline?: string
  buttonText: string
  buttonUrl: string
  urgency?: string
}

export type StepContent = QuizContent | FormContent | ContentStep | CTAContent

export interface BranchingRule {
  condition: {
    stepId: string
    operator: 'equals' | 'contains' | 'greater_than' | 'less_than'
    value: any
  }
  nextStepId: string
}

export interface Funnel {
  id: string
  name: string
  slug: string
  description?: string
  steps: FunnelStep[]
  settings: FunnelSettings
  createdAt: string
  updatedAt: string
  status: 'draft' | 'active' | 'archived'
}

export interface FunnelSettings {
  theme: {
    primaryColor: string
    fontFamily?: string
    borderRadius?: string
  }
  tracking?: {
    googleAnalytics?: string
    facebookPixel?: string
  }
  exitIntent?: boolean
  progressBar?: boolean
}

export interface Lead {
  id: string
  funnelId: string
  email: string
  data: Record<string, any>
  source?: string
  completed: boolean
  createdAt: string
}

export interface StepResponse {
  leadId: string
  stepId: string
  response: any
  duration: number
  timestamp: string
}

export interface Analytics {
  funnelId: string
  totalViews: number
  totalLeads: number
  conversionRate: number
  averageTime: number
  dropOffPoints: {
    stepId: string
    stepTitle: string
    dropOffRate: number
  }[]
  dailyStats: {
    date: string
    views: number
    leads: number
    conversions: number
  }[]
}