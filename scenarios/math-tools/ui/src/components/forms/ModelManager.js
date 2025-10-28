import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useMemo, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Brain, Loader2, Plus, Trash2 } from 'lucide-react';
import toast from 'react-hot-toast';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { formatTimestamp, safeJsonParse, stringify } from '@/lib/utils';
import { createModel, deleteModel, listModels } from '@/lib/api';
const MODEL_TYPES = ['linear_regression', 'polynomial', 'optimization', 'differential_equation'];
export function ModelManager({ settings }) {
    const queryClient = useQueryClient();
    const queryKey = useMemo(() => ['math-tools', 'models', settings.baseUrl, settings.token], [settings]);
    const { data, isLoading, isError, refetch, isFetching } = useQuery({
        queryKey,
        queryFn: () => listModels(settings),
        refetchOnWindowFocus: false,
    });
    const [isDialogOpen, setIsDialogOpen] = useState(false);
    const [name, setName] = useState('Quadratic Curve Fit');
    const [modelType, setModelType] = useState('linear_regression');
    const [formula, setFormula] = useState('y = ax^2 + bx + c');
    const [parameters, setParameters] = useState('{"a": 1.2, "b": -3.4, "c": 2.2}');
    const [isSubmitting, setIsSubmitting] = useState(false);
    async function handleCreate(event) {
        event.preventDefault();
        const parsedParameters = parameters.trim() ? safeJsonParse(parameters) : {};
        if (parameters.trim() && parsedParameters === undefined) {
            toast.error('Parameters must be valid JSON');
            return;
        }
        try {
            setIsSubmitting(true);
            await createModel({
                name,
                model_type: modelType,
                formula,
                parameters: parsedParameters,
            }, settings);
            toast.success('Model registered');
            setIsDialogOpen(false);
            setName('Quadratic Curve Fit');
            setFormula('y = ax^2 + bx + c');
            setParameters('{"a": 1.2, "b": -3.4, "c": 2.2}');
            queryClient.invalidateQueries({ queryKey });
        }
        catch (error) {
            console.error(error);
            toast.error(error instanceof Error ? error.message : 'Failed to create model');
        }
        finally {
            setIsSubmitting(false);
        }
    }
    async function handleDelete(id) {
        try {
            await deleteModel(id, settings);
            toast.success('Model removed');
            refetch();
        }
        catch (error) {
            console.error(error);
            toast.error(error instanceof Error ? error.message : 'Failed to delete model');
        }
    }
    const models = data ?? [];
    return (_jsxs(Card, { className: "h-full", children: [_jsxs(CardHeader, { className: "flex flex-col gap-3", children: [_jsxs(Badge, { variant: "secondary", className: "w-fit gap-1", children: [_jsx(Brain, { className: "h-4 w-4" }), " Model Registry"] }), _jsx(CardTitle, { children: "Curated Models Library" }), _jsx(CardDescription, { children: "Register reusable analytical models that the Math Tools API can execute on demand." })] }), _jsxs(CardContent, { className: "space-y-6", children: [_jsxs("div", { className: "flex flex-col gap-3 md:flex-row md:items-center md:justify-between", children: [_jsx("div", { className: "text-sm text-muted-foreground", children: isFetching ? 'Refreshing models…' : `${models.length} models available` }), _jsxs(Dialog, { open: isDialogOpen, onOpenChange: setIsDialogOpen, children: [_jsx(DialogTrigger, { asChild: true, children: _jsxs(Button, { size: "sm", className: "gap-2", children: [_jsx(Plus, { className: "h-4 w-4" }), " New Model"] }) }), _jsxs(DialogContent, { children: [_jsx(DialogHeader, { children: _jsx(DialogTitle, { children: "Add Mathematical Model" }) }), _jsxs("form", { onSubmit: handleCreate, className: "space-y-4", children: [_jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "model-name", children: "Model Name" }), _jsx(Input, { id: "model-name", value: name, onChange: event => setName(event.target.value), placeholder: "Quadratic Curve Fit", required: true })] }), _jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "model-type", children: "Model Type" }), _jsx(Input, { id: "model-type", value: modelType, onChange: event => setModelType(event.target.value), list: "model-types", required: true }), _jsx("datalist", { id: "model-types", children: MODEL_TYPES.map(type => (_jsx("option", { value: type }, type))) })] }), _jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "model-formula", children: "Formula" }), _jsx(Textarea, { id: "model-formula", value: formula, onChange: event => setFormula(event.target.value), rows: 3, spellCheck: false })] }), _jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "model-parameters", children: "Parameters (JSON)" }), _jsx(Textarea, { id: "model-parameters", value: parameters, onChange: event => setParameters(event.target.value), rows: 4, spellCheck: false })] }), _jsx(DialogFooter, { children: _jsxs(Button, { type: "submit", disabled: isSubmitting, className: "gap-2", children: [isSubmitting ? _jsx(Loader2, { className: "h-4 w-4 animate-spin" }) : _jsx(Plus, { className: "h-4 w-4" }), "Save Model"] }) })] })] })] })] }), _jsxs("div", { className: "space-y-3", children: [isLoading && (_jsxs("div", { className: "flex items-center gap-2 text-sm text-muted-foreground", children: [_jsx(Loader2, { className: "h-4 w-4 animate-spin" }), " Loading models\u2026"] })), isError && _jsx("p", { className: "text-sm text-rose-400", children: "Unable to load models from the persistence store." }), !isLoading && !isError && models.length === 0 && (_jsx("div", { className: "rounded-lg border border-slate-800 bg-slate-900/50 p-6 text-center text-sm text-muted-foreground", children: "No models saved yet. Create your first analytical model to unlock reusable insights." })), models.map(model => (_jsxs("div", { className: "flex flex-col gap-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm md:flex-row md:items-center md:justify-between", children: [_jsxs("div", { className: "space-y-1", children: [_jsxs("div", { className: "flex items-center gap-3", children: [_jsx("span", { className: "text-base font-semibold text-foreground", children: model.name }), _jsx(Badge, { variant: "outline", className: "capitalize", children: model.model_type.replace(/_/g, ' ') })] }), _jsxs("div", { className: "text-xs text-muted-foreground", children: ["Created ", formatTimestamp(model.created_at)] })] }), _jsx("div", { className: "flex items-center gap-2", children: _jsx(Button, { type: "button", variant: "ghost", size: "sm", className: "text-rose-300 hover:text-rose-200", onClick: () => handleDelete(model.id), children: _jsx(Trash2, { className: "h-4 w-4" }) }) })] }, model.id)))] }), models.length > 0 && (_jsxs("details", { className: "rounded-lg border border-slate-800 bg-slate-900/50", children: [_jsx("summary", { className: "cursor-pointer select-none px-4 py-3 text-sm font-semibold text-foreground", children: "API Payload Reference" }), _jsx("pre", { className: "max-h-60 overflow-auto bg-slate-950/70 p-4 text-xs text-slate-200", children: stringify({
                                    endpoint: '/api/v1/models',
                                    payload: {
                                        name: 'Model name',
                                        model_type: 'linear_regression',
                                        formula: 'y = ax + b',
                                        parameters: { a: 1.2, b: -0.4 },
                                    },
                                }, '—') })] }))] })] }));
}
