export interface Campaign {
  id: string
  name: string
  description: string
  color: string
  icon: string
  is_favorite: boolean
  prompt_count: number
  created_at: string
  updated_at: string
  metadata?: Record<string, any>
}

export interface Prompt {
  id: string
  campaign_id: string
  title: string
  content: string
  description?: string
  variables?: string[]
  tags?: string[]
  usage_count: number
  effectiveness_score?: number
  is_favorite: boolean
  word_count?: number
  estimated_tokens?: number
  created_at: string
  updated_at: string
  metadata?: Record<string, any>
}

export interface PromptTestRequest {
  model: string
  variables: Record<string, string>
}

export interface PromptTestResponse {
  response: string
  tokens_used?: number
  response_time?: number
  model_used: string
}

export interface SearchFilters {
  query?: string
  campaign_id?: string
  tags?: string[]
  is_favorite?: boolean
  limit?: number
  offset?: number
}

export interface ApiConfig {
  apiUrl: string
  uiPort: string
  resources: Record<string, string>
}

export type Theme = 'light' | 'dark' | 'system'

export interface AppSettings {
  theme: Theme
  sidebarCollapsed: boolean
  editorSettings: {
    fontSize: number
    tabSize: number
    wordWrap: boolean
    minimap: boolean
  }
}