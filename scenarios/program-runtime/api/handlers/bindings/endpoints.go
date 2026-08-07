package bindings

import "program-runtime/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{ID: "bindings_list", Path: "/vrooli.program_runtime.v1.bindings.BindingRegistryService/ListBindings", Method: "POST", Summary: "List governed manifest-bound callables.", Category: "bindings"},
	{ID: "bindings_unbound", Path: "/vrooli.program_runtime.v1.bindings.BindingRegistryService/ListUnbound", Method: "POST", Summary: "List fleet capabilities with declared unbound reasons.", Category: "bindings"},
	{ID: "bindings_act", Path: "/vrooli.program_runtime.v1.bindings.BindingRegistryService/ResolveActCells", Method: "POST", Summary: "Resolve Act operation classes against the live callable registry.", Category: "bindings"},
	{ID: "bindings_doctor", Path: "/vrooli.program_runtime.v1.bindings.BindingRegistryService/DoctorBindings", Method: "POST", Summary: "Report callable-binding health and unresolved arguments.", Category: "bindings"},
	{ID: "bindings_describe", Path: "/vrooli.program_runtime.v1.bindings.BindingRegistryService/DescribeBinding", Method: "POST", Summary: "Describe resolved proto paths for one binding.", Category: "bindings"},
}
