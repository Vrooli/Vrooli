import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Save } from "lucide-react";
import { fetchBrand, createBrand, updateBrand, type Brand } from "../lib/api";
import { Button } from "../components/ui/button";
import { Input, Textarea } from "../components/ui/input";
import { Section } from "../components/ui/section";
import { ErrorAlert } from "../components/ui/error-alert";
import { GenerateOptions } from "../components/generate-options";

// [REQ:BM-REQ-CRUD-CREATE] [REQ:BM-REQ-CRUD-UPDATE] [REQ:BM-REQ-UI-DASHBOARD] [REQ:BM-REQ-UI-GENERATE]

interface BrandFormPageProps {
  brandId?: string; // undefined = create, string = edit
  onNavigate: (path: string) => void;
}

interface FormState {
  name: string;
  description: string;
  // Identity
  display_name: string;
  tagline: string;
  // Colors
  primary: string;
  secondary: string;
  accent: string;
  background: string;
  surface: string;
  text: string;
  error_color: string;
  // Typography
  heading_font: string;
  body_font: string;
  mono_font: string;
  // Voice
  tone: string;
  style: string;
  keywords: string;
}

const emptyForm: FormState = {
  name: "", description: "",
  display_name: "", tagline: "",
  primary: "#1a365d", secondary: "#2d3748", accent: "#e53e3e",
  background: "#ffffff", surface: "#f7fafc", text: "#1a202c", error_color: "#e53e3e",
  heading_font: "", body_font: "", mono_font: "",
  tone: "", style: "", keywords: "",
};

function brandToForm(brand: Brand): FormState {
  return {
    name: brand.name,
    description: brand.description || "",
    display_name: brand.identity?.display_name || "",
    tagline: brand.identity?.tagline || "",
    primary: brand.colors?.primary || "",
    secondary: brand.colors?.secondary || "",
    accent: brand.colors?.accent || "",
    background: brand.colors?.background || "",
    surface: brand.colors?.surface || "",
    text: brand.colors?.text || "",
    error_color: brand.colors?.error || "",
    heading_font: brand.typography?.heading_font || "",
    body_font: brand.typography?.body_font || "",
    mono_font: brand.typography?.mono_font || "",
    tone: brand.voice?.tone || "",
    style: brand.voice?.style || "",
    keywords: brand.voice?.keywords?.join(", ") || "",
  };
}

function formToBrand(form: FormState): Partial<Brand> {
  return {
    name: form.name,
    description: form.description || undefined,
    identity: (form.display_name || form.tagline) ? {
      display_name: form.display_name || undefined,
      tagline: form.tagline || undefined,
    } : undefined,
    colors: {
      primary: form.primary || undefined,
      secondary: form.secondary || undefined,
      accent: form.accent || undefined,
      background: form.background || undefined,
      surface: form.surface || undefined,
      text: form.text || undefined,
      error: form.error_color || undefined,
    },
    typography: (form.heading_font || form.body_font || form.mono_font) ? {
      heading_font: form.heading_font || undefined,
      body_font: form.body_font || undefined,
      mono_font: form.mono_font || undefined,
    } : undefined,
    voice: (form.tone || form.style || form.keywords) ? {
      tone: form.tone || undefined,
      style: form.style || undefined,
      keywords: form.keywords ? form.keywords.split(",").map(k => k.trim()).filter(Boolean) : undefined,
    } : undefined,
  };
}

/** Reusable label for form fields */
function FieldLabel({ children, required }: { children: React.ReactNode; required?: boolean }) {
  return (
    <label className="block text-xs text-slate-500 mb-1">
      {children}{required && " *"}
    </label>
  );
}

export default function BrandFormPage({ brandId, onNavigate }: BrandFormPageProps) {
  const isEdit = !!brandId;
  const queryClient = useQueryClient();
  const [form, setForm] = useState<FormState>(emptyForm);
  const [submitError, setSubmitError] = useState<Error | null>(null);

  const { data: existing } = useQuery({
    queryKey: ["brand", brandId],
    queryFn: () => fetchBrand(brandId ?? ""),
    enabled: isEdit,
  });

  useEffect(() => {
    if (existing) setForm(brandToForm(existing));
  }, [existing]);

  const mutation = useMutation({
    mutationFn: (data: Partial<Brand>) =>
      isEdit && brandId ? updateBrand(brandId, data) : createBrand(data),
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: ["brands"] });
      queryClient.invalidateQueries({ queryKey: ["brand", result.id] });
      onNavigate(`/brands/${result.id}`);
    },
    onError: (err: Error) => {
      setSubmitError(err);
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.name.trim()) {
      setSubmitError(new Error("Name is required"));
      return;
    }
    setSubmitError(null);
    mutation.mutate(formToBrand(form));
  };

  const set = (key: keyof FormState) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
    setForm((prev) => ({ ...prev, [key]: e.target.value }));

  return (
    <div data-testid="brand-form-page">
      <button
        onClick={() => onNavigate(isEdit ? `/brands/${brandId}` : "/brands")}
        className="flex items-center gap-2 text-sm text-slate-400 hover:text-slate-50 mb-4 transition-colors"
        data-testid="back-from-form"
      >
        <ArrowLeft className="h-4 w-4" />
        {isEdit ? "Back to Brand" : "Back to Library"}
      </button>

      <h1 className="text-2xl font-bold text-slate-50 mb-6">
        {isEdit ? "Edit Brand" : "Create Brand"}
      </h1>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Basic Info */}
        <Section title="Basic Info">
          <div className="space-y-3">
            <div>
              <FieldLabel required>Name</FieldLabel>
              <Input
                value={form.name} onChange={set("name")}
                data-testid="brand-name-input"
                placeholder="My Brand"
              />
            </div>
            <div>
              <FieldLabel>Description</FieldLabel>
              <Textarea
                value={form.description} onChange={set("description")}
                data-testid="brand-description-input"
                rows={2}
                placeholder="A brief description of this brand..."
              />
            </div>
          </div>
        </Section>

        {/* Generation Options [REQ:BM-REQ-UI-GENERATE] */}
        {!isEdit && (
          <GenerateOptions />
        )}

        {/* Identity */}
        <Section title="Identity">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <FieldLabel>Display Name</FieldLabel>
              <Input value={form.display_name} onChange={set("display_name")} data-testid="brand-display-name-input" />
            </div>
            <div>
              <FieldLabel>Tagline</FieldLabel>
              <Input value={form.tagline} onChange={set("tagline")} data-testid="brand-tagline-input" />
            </div>
          </div>
        </Section>

        {/* Colors */}
        <Section title="Colors">
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
            {(["primary", "secondary", "accent", "background", "surface", "text", "error_color"] as const).map((key) => (
              <div key={key} className="flex items-center gap-2">
                <input
                  type="color"
                  value={form[key] || "#000000"}
                  onChange={set(key)}
                  className="h-8 w-8 rounded border border-white/20 bg-transparent cursor-pointer shrink-0"
                  data-testid={`color-picker-${key}`}
                />
                <div className="min-w-0 flex-1">
                  <label className="block text-xs text-slate-500 capitalize">{key.replace("_", " ")}</label>
                  <input
                    value={form[key]}
                    onChange={set(key)}
                    data-testid={`color-input-${key}`}
                    className="w-full text-xs font-mono bg-transparent text-slate-300 border-none outline-none"
                    placeholder="#000000"
                  />
                </div>
              </div>
            ))}
          </div>
        </Section>

        {/* Typography */}
        <Section title="Typography">
          <div className="grid grid-cols-3 gap-3">
            <div>
              <FieldLabel>Heading Font</FieldLabel>
              <Input value={form.heading_font} onChange={set("heading_font")} data-testid="brand-heading-font-input" placeholder="Inter" />
            </div>
            <div>
              <FieldLabel>Body Font</FieldLabel>
              <Input value={form.body_font} onChange={set("body_font")} data-testid="brand-body-font-input" placeholder="Inter" />
            </div>
            <div>
              <FieldLabel>Mono Font</FieldLabel>
              <Input value={form.mono_font} onChange={set("mono_font")} data-testid="brand-mono-font-input" placeholder="JetBrains Mono" />
            </div>
          </div>
        </Section>

        {/* Voice */}
        <Section title="Voice">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <FieldLabel>Tone</FieldLabel>
              <Input value={form.tone} onChange={set("tone")} data-testid="brand-tone-input" placeholder="Professional, friendly" />
            </div>
            <div>
              <FieldLabel>Style</FieldLabel>
              <Input value={form.style} onChange={set("style")} data-testid="brand-style-input" placeholder="Concise, technical" />
            </div>
          </div>
          <div className="mt-3">
            <FieldLabel>Keywords (comma-separated)</FieldLabel>
            <Input value={form.keywords} onChange={set("keywords")} data-testid="brand-keywords-input" placeholder="innovation, quality, trust" />
          </div>
        </Section>

        {submitError && (
          <ErrorAlert
            error={submitError}
            fallbackMessage="Failed to save brand."
            testId="form-error"
          />
        )}

        <div className="flex justify-end">
          <Button type="submit" disabled={mutation.isPending} data-testid="save-brand-btn">
            <Save className="mr-2 h-4 w-4" />
            {mutation.isPending ? "Saving..." : isEdit ? "Update Brand" : "Create Brand"}
          </Button>
        </div>
      </form>
    </div>
  );
}
