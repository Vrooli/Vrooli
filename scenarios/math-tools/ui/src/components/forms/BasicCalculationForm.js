import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from 'react';
import { Calculator, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { parseNumberList, stringify } from '@/lib/utils';
import { performCalculation } from '@/lib/api';
import toast from 'react-hot-toast';
const BASIC_OPERATIONS = [
    { value: 'add', label: 'Addition (Σ)' },
    { value: 'subtract', label: 'Subtraction' },
    { value: 'multiply', label: 'Multiplication (Π)' },
    { value: 'divide', label: 'Division' },
    { value: 'power', label: 'Power' },
    { value: 'sqrt', label: 'Square Root' },
    { value: 'log', label: 'Logarithm' },
    { value: 'exp', label: 'Exponential' },
];
export function BasicCalculationForm({ settings }) {
    const [operation, setOperation] = useState('add');
    const [dataInput, setDataInput] = useState('2, 4, 6, 8, 10');
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [response, setResponse] = useState(null);
    async function handleSubmit(event) {
        event.preventDefault();
        const data = parseNumberList(dataInput);
        if (data.length === 0) {
            toast.error('Please provide at least one numeric value');
            return;
        }
        try {
            setIsSubmitting(true);
            const result = await performCalculation({
                operation,
                data,
            }, settings);
            setResponse(result);
            toast.success('Calculation complete');
        }
        catch (error) {
            console.error(error);
            toast.error(error instanceof Error ? error.message : 'Failed to execute calculation');
        }
        finally {
            setIsSubmitting(false);
        }
    }
    return (_jsxs(Card, { className: "h-full", children: [_jsxs(CardHeader, { className: "flex flex-col gap-3", children: [_jsxs(Badge, { variant: "secondary", className: "w-fit gap-1", children: [_jsx(Calculator, { className: "h-4 w-4" }), " Basic Arithmetic"] }), _jsx(CardTitle, { children: "Run Quick Calculations" }), _jsx(CardDescription, { children: "Evaluate core arithmetic operations with arbitrary precision." })] }), _jsxs(CardContent, { children: [_jsxs("form", { onSubmit: handleSubmit, className: "space-y-4", children: [_jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "basic-operation", children: "Operation" }), _jsxs(Select, { value: operation, onValueChange: setOperation, children: [_jsx(SelectTrigger, { id: "basic-operation", children: _jsx(SelectValue, { placeholder: "Choose operation" }) }), _jsx(SelectContent, { children: BASIC_OPERATIONS.map(option => (_jsx(SelectItem, { value: option.value, children: option.label }, option.value))) })] })] }), _jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "basic-values", children: "Values" }), _jsx(Input, { id: "basic-values", value: dataInput, onChange: event => setDataInput(event.target.value), placeholder: "Comma or space separated numbers", autoComplete: "off" })] }), _jsx(Button, { type: "submit", className: "w-full", disabled: isSubmitting, children: isSubmitting ? (_jsxs("span", { className: "flex items-center gap-2", children: [_jsx(Loader2, { className: "h-4 w-4 animate-spin" }), "Calculating\u2026"] })) : ('Calculate') })] }), response && (_jsxs("div", { className: "mt-6 space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm", children: [_jsxs("div", { className: "flex items-center justify-between text-xs uppercase tracking-wide text-muted-foreground", children: [_jsx("span", { children: response.operation }), _jsxs("span", { children: [response.execution_time_ms, " ms"] })] }), _jsx("pre", { className: "max-h-52 overflow-auto rounded-md bg-slate-950/70 p-3 text-xs", children: stringify(response.result) })] }))] })] }));
}
