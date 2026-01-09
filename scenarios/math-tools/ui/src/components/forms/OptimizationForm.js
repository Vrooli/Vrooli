import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from 'react';
import { Goal, Loader2 } from 'lucide-react';
import toast from 'react-hot-toast';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import { safeJsonParse, stringify } from '@/lib/utils';
import { optimize } from '@/lib/api';
export function OptimizationForm({ settings }) {
    const [objectiveFunction, setObjectiveFunction] = useState('x^2 + y^2');
    const [variables, setVariables] = useState('x, y');
    const [optimizationType, setOptimizationType] = useState('minimize');
    const [algorithm, setAlgorithm] = useState('gradient_descent');
    const [boundsInput, setBoundsInput] = useState('{"x": [-5, 5], "y": [-5, 5]}');
    const [tolerance, setTolerance] = useState('0.0001');
    const [maxIterations, setMaxIterations] = useState('500');
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [response, setResponse] = useState(null);
    async function handleSubmit(event) {
        event.preventDefault();
        const parsedVariables = variables
            .split(/[\s,]+/u)
            .map(value => value.trim())
            .filter(Boolean);
        if (parsedVariables.length === 0) {
            toast.error('Provide at least one variable');
            return;
        }
        let parsedBounds;
        if (boundsInput.trim()) {
            parsedBounds = safeJsonParse(boundsInput);
            if (!parsedBounds) {
                toast.error('Bounds must be valid JSON, e.g. {"x":[-5,5]}');
                return;
            }
        }
        try {
            setIsSubmitting(true);
            const result = await optimize({
                objective_function: objectiveFunction,
                variables: parsedVariables,
                optimization_type: optimizationType,
                algorithm,
                options: {
                    bounds: parsedBounds,
                    tolerance: Number.parseFloat(tolerance) || 1e-6,
                    max_iterations: Number.parseInt(maxIterations, 10) || 500,
                },
            }, settings);
            setResponse(result);
            toast.success('Optimization complete');
        }
        catch (error) {
            console.error(error);
            toast.error(error instanceof Error ? error.message : 'Optimization failed');
        }
        finally {
            setIsSubmitting(false);
        }
    }
    return (_jsxs(Card, { className: "h-full", children: [_jsxs(CardHeader, { className: "flex flex-col gap-3", children: [_jsxs(Badge, { variant: "secondary", className: "w-fit gap-1", children: [_jsx(Goal, { className: "h-4 w-4" }), " Optimization"] }), _jsx(CardTitle, { children: "Optimization Engine" }), _jsx(CardDescription, { children: "Find optimal solutions with gradient-based search and bounds awareness." })] }), _jsxs(CardContent, { children: [_jsxs("form", { onSubmit: handleSubmit, className: "space-y-4", children: [_jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "objective", children: "Objective Function" }), _jsx(Textarea, { id: "objective", value: objectiveFunction, onChange: event => setObjectiveFunction(event.target.value), spellCheck: false })] }), _jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "variables", children: "Variables" }), _jsx(Input, { id: "variables", value: variables, onChange: event => setVariables(event.target.value), placeholder: "x, y" })] }), _jsxs("div", { className: "grid gap-3 md:grid-cols-3", children: [_jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "optimization-type", children: "Optimization Type" }), _jsxs(Select, { value: optimizationType, onValueChange: value => setOptimizationType(value), children: [_jsx(SelectTrigger, { id: "optimization-type", children: _jsx(SelectValue, { placeholder: "Select" }) }), _jsxs(SelectContent, { children: [_jsx(SelectItem, { value: "minimize", children: "Minimize" }), _jsx(SelectItem, { value: "maximize", children: "Maximize" })] })] })] }), _jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "algorithm", children: "Algorithm" }), _jsx(Input, { id: "algorithm", value: algorithm, onChange: event => setAlgorithm(event.target.value), placeholder: "gradient_descent" })] }), _jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "tolerance", children: "Tolerance" }), _jsx(Input, { id: "tolerance", value: tolerance, onChange: event => setTolerance(event.target.value) })] })] }), _jsxs("div", { className: "grid gap-2 md:grid-cols-2", children: [_jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "max-iterations", children: "Max Iterations" }), _jsx(Input, { id: "max-iterations", value: maxIterations, onChange: event => setMaxIterations(event.target.value) })] }), _jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "bounds", children: "Variable Bounds (JSON)" }), _jsx(Textarea, { id: "bounds", value: boundsInput, onChange: event => setBoundsInput(event.target.value), spellCheck: false })] })] }), _jsx(Button, { type: "submit", className: "w-full", disabled: isSubmitting, children: isSubmitting ? (_jsxs("span", { className: "flex items-center gap-2", children: [_jsx(Loader2, { className: "h-4 w-4 animate-spin" }), "Optimizing\u2026"] })) : ('Run Optimization') })] }), response && (_jsxs("div", { className: "mt-6 space-y-4", children: [_jsxs("div", { className: "grid gap-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm md:grid-cols-2", children: [_jsxs("div", { className: "flex flex-col", children: [_jsx("span", { className: "text-xs uppercase tracking-wide text-muted-foreground", children: "Status" }), _jsx("span", { className: "text-base font-semibold text-foreground", children: response.status })] }), _jsxs("div", { className: "flex flex-col", children: [_jsx("span", { className: "text-xs uppercase tracking-wide text-muted-foreground", children: "Iterations" }), _jsx("span", { className: "text-base font-semibold text-foreground", children: response.iterations })] }), _jsxs("div", { className: "flex flex-col", children: [_jsx("span", { className: "text-xs uppercase tracking-wide text-muted-foreground", children: "Optimal Value" }), _jsx("span", { className: "text-base font-semibold text-emerald-400", children: response.optimal_value.toFixed(4) })] }), _jsxs("div", { className: "flex flex-col", children: [_jsx("span", { className: "text-xs uppercase tracking-wide text-muted-foreground", children: "Algorithm" }), _jsx("span", { className: "text-base font-semibold text-foreground", children: response.algorithm_used })] })] }), _jsxs("div", { className: "space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm", children: [_jsx("div", { className: "text-xs uppercase tracking-wide text-muted-foreground", children: "Optimal Solution" }), _jsx("pre", { className: "max-h-52 overflow-auto rounded-md bg-slate-950/70 p-3 text-xs", children: stringify(response.optimal_solution) })] }), response.sensitivity_analysis && (_jsxs("div", { className: "space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm", children: [_jsx("div", { className: "text-xs uppercase tracking-wide text-muted-foreground", children: "Sensitivity Analysis" }), _jsx("pre", { className: "max-h-52 overflow-auto rounded-md bg-slate-950/70 p-3 text-xs", children: stringify(response.sensitivity_analysis) })] }))] }))] })] }));
}
