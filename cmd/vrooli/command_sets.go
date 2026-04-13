package main

import (
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/resources"
)

type appSubcommandDescriptor struct {
	Name    string
	Group   string
	Summary string
	Hidden  bool
	Handler appCommandHandler
}

type resourceSubcommandDescriptor struct {
	Name    string
	Group   string
	Summary string
	Hidden  bool
	Handler resourceCommandHandler
}

func normalizeSubcommand(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func buildAppSubcommandMap(descriptors []appSubcommandDescriptor) map[string]appCommandHandler {
	items := make(map[string]appCommandHandler, len(descriptors))
	for _, descriptor := range descriptors {
		items[descriptor.Name] = descriptor.Handler
	}
	return items
}

func buildResourceSubcommandMap(descriptors []resourceSubcommandDescriptor) map[string]resourceCommandHandler {
	items := make(map[string]resourceCommandHandler, len(descriptors))
	for _, descriptor := range descriptors {
		items[descriptor.Name] = descriptor.Handler
	}
	return items
}

func runAppSubcommandSet(app *App, ctx *commandContext, args []string, usage func(io.Writer), command string, handlers map[string]appCommandHandler) error {
	if len(args) == 0 || wantsCommandHelp(args) {
		usage(ctx.Stdout)
		return nil
	}
	handler, ok := handlers[normalizeSubcommand(args[0])]
	if !ok {
		return usageErrorf(command, "unknown %s command: %s", command, args[0])
	}
	return handler(app, ctx, args[1:])
}

func runResourceSubcommandSet(app *App, ctx *commandContext, controller *resources.Controller, args []string, usage func(io.Writer), command string, handlers map[string]resourceCommandHandler) error {
	if len(args) == 0 || wantsCommandHelp(args) {
		usage(ctx.Stdout)
		return nil
	}
	handler, ok := handlers[normalizeSubcommand(args[0])]
	if !ok {
		return usageErrorf(command, "unknown %s command: %s", command, args[0])
	}
	return handler(app, ctx, controller, args[1:])
}

func renderSubcommandHelp(w io.Writer, title, usage, defaultGroup string, descriptors []appSubcommandDescriptor) {
	if strings.TrimSpace(title) != "" {
		_, _ = io.WriteString(w, title+"\n\n")
	}
	if strings.TrimSpace(usage) != "" {
		_, _ = io.WriteString(w, "Usage:\n")
		_, _ = io.WriteString(w, "  "+usage+"\n\n")
	}
	renderCommandGroups(w, toCommandDescriptors(descriptors, defaultGroup))
}

func renderResourceSubcommandHelp(w io.Writer, title, usage, defaultGroup string, descriptors []resourceSubcommandDescriptor) {
	if strings.TrimSpace(title) != "" {
		_, _ = io.WriteString(w, title+"\n\n")
	}
	if strings.TrimSpace(usage) != "" {
		_, _ = io.WriteString(w, "Usage:\n")
		_, _ = io.WriteString(w, "  "+usage+"\n\n")
	}
	renderCommandGroups(w, toResourceCommandDescriptors(descriptors, defaultGroup))
}

func toCommandDescriptors(items []appSubcommandDescriptor, defaultGroup string) []commandDescriptor {
	descriptors := make([]commandDescriptor, 0, len(items))
	for _, item := range items {
		if item.Hidden {
			continue
		}
		group := item.Group
		if strings.TrimSpace(group) == "" {
			group = defaultGroup
		}
		descriptors = append(descriptors, commandDescriptor{
			Name:    item.Name,
			Group:   group,
			Summary: item.Summary,
		})
	}
	return descriptors
}

func toResourceCommandDescriptors(items []resourceSubcommandDescriptor, defaultGroup string) []commandDescriptor {
	descriptors := make([]commandDescriptor, 0, len(items))
	for _, item := range items {
		if item.Hidden {
			continue
		}
		group := item.Group
		if strings.TrimSpace(group) == "" {
			group = defaultGroup
		}
		descriptors = append(descriptors, commandDescriptor{
			Name:    item.Name,
			Group:   group,
			Summary: item.Summary,
		})
	}
	return descriptors
}
