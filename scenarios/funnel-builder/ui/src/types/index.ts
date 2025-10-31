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

export interface FunnelTheme {
  primaryColor: string
  fontFamily?: string
  borderRadius?: string
  [key: string]: unknown
}

export interface FunnelSettings {
  theme: FunnelTheme
  tracking?: {
    googleAnalytics?: string
    facebookPixel?: string
    [key: string]: unknown
  }
  exitIntent?: boolean
  progressBar?: boolean
  [key: string]: unknown
}

export interface Funnel {
  id: string
  tenantId?: string | null
  name: string
  slug: string
  description?: string
  steps: FunnelStep[]
  settings: FunnelSettings
  createdAt: string
  updatedAt: string
  status: 'draft' | 'active' | 'archived'
}

export interface Lead {
  id: string
  funnelId: string
  email?: string
  phone?: string
  name?: string
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

export interface StepDropOffPoint {
  stepId: string
  stepTitle: string
  position: number
  dropOffRate: number
  responses: number
  visitors: number
  avgDuration?: number | null
}

export interface DailyStat {
  date: string
  views: number
  leads: number
  conversions: number
}

export interface TrafficSourceStat {
  source: string
  count: number
  percentage: number
}

export interface FunnelAnalytics {
  funnelId: string
  totalViews: number
  totalLeads: number
  completedLeads: number
  capturedLeads: number
  conversionRate: number
  captureRate: number
  averageTime: number | null
  dropOffPoints: StepDropOffPoint[]
  dailyStats: DailyStat[]
  trafficSources: TrafficSourceStat[]
}
