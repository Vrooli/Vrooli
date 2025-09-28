export interface ModelConfig {
  model?: string;
  temperature?: number;
  max_tokens?: number;
  [key: string]: unknown;
}

export interface WidgetConfig {
  theme?: string;
  position?: string;
  primaryColor?: string;
  greeting?: string;
  [key: string]: unknown;
}

export interface Chatbot {
  id: string;
  name: string;
  description?: string;
  personality: string;
  knowledge_base?: string;
  model_config: ModelConfig;
  widget_config: WidgetConfig;
  escalation_config?: Record<string, unknown> | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface ChatbotSummary extends Chatbot {
  tenant_id?: string;
}

export interface AnalyticsIntent {
  name: string;
  count: number;
}

export interface AnalyticsData {
  total_conversations: number;
  total_messages: number;
  leads_captured: number;
  avg_conversation_length: number;
  engagement_score: number;
  conversion_rate: number;
  top_intents: AnalyticsIntent[];
  conversion_funnel?: {
    total_visitors: number;
    engaged_visitors: number;
    qualified_leads: number;
    captured_leads: number;
    engagement_rate: number;
    qualification_rate: number;
    capture_rate: number;
  } | null;
  user_journey?: {
    avg_time_to_engagement_seconds: number;
    avg_time_to_lead_seconds: number;
  } | null;
}

export interface ApiHealthResponse {
  status: 'healthy' | 'degraded' | 'unhealthy';
  service: string;
  timestamp: string;
  readiness: boolean;
  version?: string;
  dependencies?: Record<string, unknown>;
}

export interface ApiConnectionState {
  status: 'checking' | 'healthy' | 'degraded' | 'error';
  latencyMs: number | null;
  message?: string;
  lastChecked?: string;
}
