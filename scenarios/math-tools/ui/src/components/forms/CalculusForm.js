import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from 'react';
import { Sigma, Loader2 } from 'lucide-react';
import toast from 'react-hot-toast';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { parseNumberList, stringify } from '@/lib/utils';
import { performCalculation } from '@/lib/api';
const CALCULUS_OPERATIONS = [
    { value: 'derivative', label: 'Derivative (f′)' },
    { value: 'integral', label: 'Definite Integral' },
    { value: 'partial_derivative', label: 'Partial Derivative' },
    { value: 'double_integral', label: 'Double Integral' },
];
const OPERATION_HELP = {
    derivative: 'Provide a single value for the evaluation point (e.g. x = 3).',
    integral: 'Provide lower and upper bounds (e.g. 0, 5).',
    partial_derivative: 'Provide variable value (e.g. x, y). Currently uses sample function f(x, y) = x² + y².',
    double_integral: 'Provide lower/upper bounds for x and y (e.g. 0, 1, 0, 2).',
};
export function CalculusForm({ settings }) {
    const [operation, setOperation] = useState('derivative');
    const [dataInput, setDataInput] = useState('3');
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [response, setResponse] = useState(null);
    function getPlaceholder() {
        switch (operation) {
            case 'integral':
                return 'Lower, Upper bounds (e.g. 0, 5)';
            case 'partial_derivative':
                return 'Variable values (e.g. 1.5, 2.2)';
            case 'double_integral':
                return 'x₀, x₁, y₀, y₁ (e.g. 0, 1, 0, 2)';
            default:
                return 'Evaluation point (e.g. 3)';
        }
    }
    async function handleSubmit(event) {
        event.preventDefault();
        const data = parseNumberList(dataInput);
        if (data.length === 0) {
            toast.error('Provide at least one numeric value');
            return;
        }
        try {
            setIsSubmitting(true);
            const result = await performCalculation({
                operation,
                data,
            }, settings);
            setResponse(result);
            toast.success('Calculus operation complete');
        }
        catch (error) {
            console.error(error);
            toast.error(error instanceof Error ? error.message : 'Calculus operation failed');
        }
        finally {
            setIsSubmitting(false);
        }
    }
    return (_jsxs(Card, { className: "h-full", children: [_jsxs(CardHeader, { className: "flex flex-col gap-3", children: [_jsxs(Badge, { variant: "secondary", className: "w-fit gap-1", children: [_jsx(Sigma, { className: "h-4 w-4" }), " Calculus"] }), _jsx(CardTitle, { children: "Calculus Playground" }), _jsx(CardDescription, { children: OPERATION_HELP[operation] })] }), _jsxs(CardContent, { children: [_jsxs("form", { onSubmit: handleSubmit, className: "space-y-4", children: [_jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "calculus-operation", children: "Operation" }), _jsxs(Select, { value: operation, onValueChange: value => {
                                            setOperation(value);
                                            setResponse(null);
                                            switch (value) {
                                                case 'integral':
                                                    setDataInput('0, 5');
                                                    break;
                                                case 'partial_derivative':
                                                    setDataInput('1.5, 2.2');
                                                    break;
                                                case 'double_integral':
                                                    setDataInput('0, 1, 0, 2');
                                                    break;
                                                default:
                                                    setDataInput('3');
                                            }
                                        }, children: [_jsx(SelectTrigger, { id: "calculus-operation", children: _jsx(SelectValue, { placeholder: "Choose calculus operation" }) }), _jsx(SelectContent, { children: CALCULUS_OPERATIONS.map(option => (_jsx(SelectItem, { value: option.value, children: option.label }, option.value))) })] })] }), _jsxs("div", { className: "grid gap-2", children: [_jsx(Label, { htmlFor: "calculus-data", children: "Inputs" }), _jsx(Input, { id: "calculus-data", value: dataInput, onChange: event => setDataInput(event.target.value), placeholder: getPlaceholder(), autoComplete: "off" })] }), _jsx(Button, { type: "submit", className: "w-full", disabled: isSubmitting, children: isSubmitting ? (_jsxs("span", { className: "flex items-center gap-2", children: [_jsx(Loader2, { className: "h-4 w-4 animate-spin" }), "Computing\u2026"] })) : ('Compute') })] }), response && (_jsxs("div", { className: "mt-6 space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm", children: [_jsxs("div", { className: "flex items-center justify-between text-xs uppercase tracking-wide text-muted-foreground", children: [_jsx("span", { children: response.operation }), _jsxs("span", { children: [response.execution_time_ms, " ms"] })] }), _jsx("pre", { className: "max-h-52 overflow-auto rounded-md bg-slate-950/70 p-3 text-xs", children: stringify(response.result) })] }))] })] }));
}
