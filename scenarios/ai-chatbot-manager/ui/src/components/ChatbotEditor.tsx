import { ChangeEvent, FormEvent, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  ArrowLeft,
  Cpu,
  Loader2,
  Palette,
  PenSquare,
  Rocket,
  Wand2,
} from 'lucide-react';

import apiClient from '../utils/api';
import type { Chatbot, ModelConfig, WidgetConfig } from '../types';

interface ChatbotForm {
  name: string;
  description: string;
  personality: string;
  knowledge_base: string;
  model_config: ModelConfig;
  widget_config: WidgetConfig & { greeting?: string };
}

const DEFAULT_FORM: ChatbotForm = {
  name: '',
  description: '',
  personality: 'You are a helpful assistant who responds with confidence and empathy.',
  knowledge_base: '',
  model_config: {
    model: 'llama3.2',
    temperature: 0.7,
    max_tokens: 1000,
  },
  widget_config: {
    theme: 'light',
    position: 'bottom-right',
    primaryColor: '#2563eb',
    greeting: 'Hi! How can I help you today?',
  },
};

function ChatbotEditor() {
  const { id } = useParams();
  const navigate = useNavigate();
  const isEditing = Boolean(id);

  const [formData, setFormData] = useState<ChatbotForm>(DEFAULT_FORM);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let isMounted = true;

    const loadChatbot = async () => {
      if (!isEditing || !id) return;

      try {
        setLoading(true);
        const response = await apiClient.get(`/api/v1/chatbots/${id}`);
        if (!response.ok) {
          alert('Unable to load chatbot details.');
          navigate('/chatbots');
          return;
        }

        const chatbot = (await response.json()) as Chatbot;
        if (!isMounted) return;

        setFormData({
          name: chatbot.name || DEFAULT_FORM.name,
          description: chatbot.description || DEFAULT_FORM.description,
          personality: chatbot.personality || DEFAULT_FORM.personality,
          knowledge_base: chatbot.knowledge_base || DEFAULT_FORM.knowledge_base,
          model_config: {
            model: (chatbot.model_config?.model as string) || 'llama3.2',
            temperature: (chatbot.model_config?.temperature as number) ?? 0.7,
            max_tokens: (chatbot.model_config?.max_tokens as number) ?? 1000,
          },
          widget_config: {
            theme: (chatbot.widget_config?.theme as string) || 'light',
            position: (chatbot.widget_config?.position as string) || 'bottom-right',
            primaryColor: (chatbot.widget_config?.primaryColor as string) || '#2563eb',
            greeting: (chatbot.widget_config?.greeting as string) || DEFAULT_FORM.widget_config?.greeting,
          },
        });
      } catch (error) {
        console.error('Failed to load chatbot:', error);
        alert('Unable to load chatbot details.');
        navigate('/chatbots');
      } finally {
        if (isMounted) {
          setLoading(false);
        }
      }
    };

    loadChatbot();
    return () => {
      isMounted = false;
    };
  }, [id, isEditing, navigate]);

  const handleFieldChange = (field: keyof ChatbotForm) => (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const value = event.target.value;
    setFormData((prev) => ({
      ...prev,
      [field]: value,
    }));
  };

  const handleModelConfigChange = (field: keyof ModelConfig) => (event: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const value = event.target.type === 'number' ? Number(event.target.value) : event.target.value;
    setFormData((prev) => ({
      ...prev,
      model_config: {
        ...prev.model_config,
        [field]: value,
      },
    }));
  };

  const handleWidgetConfigChange = (field: keyof (WidgetConfig & { greeting?: string })) => (
    event: ChangeEvent<HTMLInputElement | HTMLSelectElement>
  ) => {
    const value = event.target.type === 'color' ? event.target.value : event.target.value;
    setFormData((prev) => ({
      ...prev,
      widget_config: {
        ...prev.widget_config,
        [field]: value,
      },
    }));
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (!formData.name.trim()) {
      alert('Please provide a descriptive name for this chatbot.');
      return;
    }

    try {
      setSaving(true);
      const response = isEditing && id
        ? await apiClient.put(`/api/v1/chatbots/${id}`, formData)
        : await apiClient.post('/api/v1/chatbots', formData);

      if (!response.ok) {
        alert('Something went wrong while saving.');
        return;
      }

      const payload = (await response.json()) as { chatbot?: Chatbot; id?: string };
      const chatbotId = id || payload?.chatbot?.id || payload?.id;

      alert(isEditing ? 'Chatbot updated successfully.' : 'Chatbot created successfully.');

      if (!isEditing && chatbotId) {
        const embedSnippet = `<script src="${window.location.origin}/widget.js" data-chatbot-id="${chatbotId}"></script>`;
        if (window.confirm('Copy the embed snippet to deploy this chatbot?')) {
          navigator.clipboard.writeText(embedSnippet).catch(() => {
            alert('Unable to copy automatically. Please copy the snippet manually.');
          });
        }
      }

      navigate('/chatbots');
    } catch (error) {
      console.error('Failed to save chatbot:', error);
      alert('Unable to save chatbot. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  const heading = useMemo(
    () =>
      isEditing
        ? { title: 'Refine chatbot experience', subtitle: 'Adjust tone, logic, and widget presentation in minutes.' }
        : { title: 'Launch a new chatbot', subtitle: 'Craft a high-converting assistant tailored to your brand.' },
    [isEditing]
  );

  if (loading) {
    return (
      <div className="card loading-card">
        <div className="loading-spinner" />
        <p>Loading chatbot configuration…</p>
      </div>
    );
  }

  return (
    <div className="page-stack">
      <header className="page-header">
        <button className="ghost-button" onClick={() => navigate('/chatbots')}>
          <ArrowLeft size={16} />
          <span>Back to chatbots</span>
        </button>
        <div>
          <h2>{heading.title}</h2>
          <p>{heading.subtitle}</p>
        </div>
      </header>

      <form className="editor-form" onSubmit={handleSubmit}>
        <section className="form-card">
          <header>
            <div>
              <h3>
                <PenSquare size={18} /> Identity & position
              </h3>
              <p>Define the persona, promise, and strategic positioning of this assistant.</p>
            </div>
          </header>
          <div className="form-grid">
            <div className="form-control">
              <label htmlFor="chatbot-name">Chatbot name *</label>
              <input
                id="chatbot-name"
                type="text"
                value={formData.name}
                onChange={handleFieldChange('name')}
                placeholder="e.g. Enterprise Success Concierge"
                required
              />
              <span className="form-hint">Appears in analytics and internal dashboards.</span>
            </div>
            <div className="form-control">
              <label htmlFor="chatbot-description">Description</label>
              <input
                id="chatbot-description"
                type="text"
                value={formData.description}
                onChange={handleFieldChange('description')}
                placeholder="High-touch onboarding companion for enterprise clients"
              />
              <span className="form-hint">Short summary to help your team remember the goal for this bot.</span>
            </div>
            <div className="form-control full">
              <label htmlFor="chatbot-personality">Personality & instructions *</label>
              <textarea
                id="chatbot-personality"
                value={formData.personality}
                onChange={handleFieldChange('personality')}
                rows={6}
                required
              />
              <span className="form-hint">Outline tone, guardrails, escalation rules, and value proposition.</span>
            </div>
            <div className="form-control full">
              <label htmlFor="chatbot-knowledge">Knowledge base</label>
              <textarea
                id="chatbot-knowledge"
                value={formData.knowledge_base}
                onChange={handleFieldChange('knowledge_base')}
                rows={6}
                placeholder="Key products, SLAs, pricing objections, integration requirements…"
              />
              <span className="form-hint">Provide structured context the chatbot can reference when answering questions.</span>
            </div>
          </div>
        </section>

        <section className="form-card">
          <header>
            <div>
              <h3>
                <Cpu size={18} /> AI configuration
              </h3>
              <p>Align generative behavior with business outcomes and compliance policies.</p>
            </div>
          </header>
          <div className="form-grid">
            <div className="form-control">
              <label htmlFor="model-select">Model</label>
              <select id="model-select" value={formData.model_config?.model as string} onChange={handleModelConfigChange('model')}>
                <option value="llama3.2">Llama 3.2 (recommended)</option>
                <option value="mistral">Mistral</option>
                <option value="codellama">CodeLlama</option>
              </select>
            </div>
            <div className="form-control">
              <label htmlFor="temperature">Temperature</label>
              <input
                id="temperature"
                type="number"
                min={0}
                max={1}
                step={0.1}
                value={formData.model_config?.temperature ?? 0.7}
                onChange={handleModelConfigChange('temperature')}
              />
              <span className="form-hint">Controls creativity (0 = precise, 1 = imaginative).</span>
            </div>
            <div className="form-control">
              <label htmlFor="tokens">Max tokens</label>
              <input
                id="tokens"
                type="number"
                min={100}
                max={4000}
                step={100}
                value={formData.model_config?.max_tokens ?? 1000}
                onChange={handleModelConfigChange('max_tokens')}
              />
              <span className="form-hint">Set the maximum response length for each answer.</span>
            </div>
          </div>
        </section>

        <section className="form-card">
          <header>
            <div>
              <h3>
                <Palette size={18} /> Widget experience
              </h3>
              <p>Tailor the on-site assistant to match brand guidelines and placement.</p>
            </div>
          </header>
          <div className="form-grid">
            <div className="form-control">
              <label htmlFor="widget-theme">Theme</label>
              <select id="widget-theme" value={formData.widget_config?.theme as string} onChange={handleWidgetConfigChange('theme')}>
                <option value="light">Light</option>
                <option value="dark">Dark</option>
                <option value="professional">Professional</option>
                <option value="modern">Modern</option>
              </select>
            </div>
            <div className="form-control">
              <label htmlFor="widget-position">Position</label>
              <select id="widget-position" value={formData.widget_config?.position as string} onChange={handleWidgetConfigChange('position')}>
                <option value="bottom-right">Bottom right</option>
                <option value="bottom-left">Bottom left</option>
                <option value="center">Center</option>
              </select>
            </div>
            <div className="form-control">
              <label htmlFor="primary-color">Primary color</label>
              <input
                id="primary-color"
                type="color"
                value={(formData.widget_config?.primaryColor as string) || '#2563eb'}
                onChange={handleWidgetConfigChange('primaryColor')}
              />
            </div>
            <div className="form-control full">
              <label htmlFor="widget-greeting">Greeting message</label>
              <input
                id="widget-greeting"
                type="text"
                value={formData.widget_config?.greeting as string}
                onChange={handleWidgetConfigChange('greeting')}
                placeholder="Hi! How can I help you today?"
              />
              <span className="form-hint">Displayed as soon as a visitor opens the widget.</span>
            </div>
          </div>
        </section>

        <footer className="form-footer">
          <button type="button" className="ghost-button" onClick={() => navigate('/chatbots')} disabled={saving}>
            <Rocket size={16} />
            <span>Cancel</span>
          </button>
          <button type="submit" className="primary-action" disabled={saving}>
            {saving ? (
              <>
                <Loader2 size={16} className="spin" />
                <span>{isEditing ? 'Saving changes' : 'Create chatbot'}</span>
              </>
            ) : (
              <>
                <Wand2 size={16} />
                <span>{isEditing ? 'Save changes' : 'Create chatbot'}</span>
              </>
            )}
          </button>
        </footer>
      </form>
    </div>
  );
}

export default ChatbotEditor;
