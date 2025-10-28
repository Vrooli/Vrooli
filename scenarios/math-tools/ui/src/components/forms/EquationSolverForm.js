import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from 'react';
import { Braces, Loader2 } from 'lucide-react';
import toast from 'react-hot-toast';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { stringify } from '@/lib/utils';
import { solveEquation } from '@/lib/api';
export function EquationSolverForm({ settings }) {
    const [equation, setEquation] = useState('x^2 - 4 = 0');
    const [variables, setVariables] = useState('x');
    const [method, setMethod] = useState('numerical');
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [response, setResponse] = useState(null);
    async function handleSubmit(event) {
        event.preventDefault();
        const parsedVariables = variables
            .split(/[\s,]+/u)
            .map(value => value.trim())
            .filter(Boolean);
        if (parsedVariables.length === 0) {
            toast.error('Provide at least one variable name');
            return;
        }
        try {
            setIsSubmitting(true);
            const result = await solveEquation({
                equations: equation,
                variables: parsedVariables,
                method,
                options: {
                    tolerance: 1e-6,
                    max_iterations: 1000,
                },
            }, settings);
            setResponse(result);
            toast.success('Equation solved');
        }
        catch (error) {
            console.error(error);
            toast.error(error instanceof Error ? error.message : 'Equation solving failed');
        }
        finally {
            setIsSubmitting(false);
        }
    }
    return (_jsxs(Card, { className: "h-full", children: [_jsxs(CardHeader, { className: "flex flex-col gap-3", children: [_jsxs(Badge, { variant: "secondary", className: "w-fit gap-1", children: [_jsx(Braces, { className: "h-4 w-4" }), " Equation Solver"] }), _jsx(CardTitle, { children: "Non-linear Equation Solver" }), _jsx(CardDescription, { children: "Use Newton-Raphson and analytical shortcuts to solve equations quickly." })] }), _jsxs(CardContent, { children: [_jsxs("form", { onSubmit: handleSubmit, className: "space-y-4", children: [_jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "equation-expression", children: "Equation" }), _jsx(Input, { id: "equation-expression", value: equation, onChange: event => setEquation(event.target.value), placeholder: "x^2 - 4 = 0", autoComplete: "off" })] }), _jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "equation-variables", children: "Variables" }), _jsx(Input, { id: "equation-variables", value: variables, onChange: event => setVariables(event.target.value), placeholder: "x, y", autoComplete: "off" })] }), _jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "equation-method", children: "Method" }), _jsx(Input, { id: "equation-method", value: method, onChange: event => setMethod(event.target.value), placeholder: "numerical", autoComplete: "off" })] }), _jsx(Button, { type: "submit", className: "w-full", disabled: isSubmitting, children: isSubmitting ? (_jsxs("span", { className: "flex items-center gap-2", children: [_jsx(Loader2, { className: "h-4 w-4 animate-spin" }), "Solving\u2026"] })) : ('Solve Equation') })] }), response && (_jsxs("div", { className: "mt-6 space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm", children: [_jsxs("div", { className: "grid gap-2 text-xs uppercase tracking-wide text-muted-foreground", children: [_jsxs("div", { className: "flex justify-between", children: [_jsx("span", { children: "Method" }), _jsx("span", { children: response.method_used })] }), _jsxs("div", { className: "flex justify-between", children: [_jsx("span", { children: "Solution Type" }), _jsx("span", { children: response.solution_type })] }), _jsxs("div", { className: "flex justify-between", children: [_jsx("span", { children: "Iterations" }), _jsx("span", { children: response.convergence_info.iterations })] })] }), _jsx("pre", { className: "max-h-52 overflow-auto rounded-md bg-slate-950/70 p-3 text-xs", children: stringify(response.solutions) })] }))] })] }));
}
