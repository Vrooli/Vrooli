import { Meta, Story } from "@storybook/react";
import { ActorView, OrganizationView, ProjectView, RoutineView, StandardView } from "components";
import { BaseObjectDialog as Component } from '..';
import { BaseObjectDialogProps as Props } from '../types';

// Define story metadata
export default {
    title: 'dialogs/BaseObjectDialog',
    component: Component,
} as Meta;

// Define templates for enabling control over props
const EmptyTemplate: Story<Props> = (args) => <Component {...args} />;
const ActorTemplate: Story<Props> = (args) => (<Component {...args}>
    <ActorView session={{}} />
</Component>);
const OrganizationTemplate: Story<Props> = (args) => (<Component {...args}>
    <OrganizationView session={{}} />
</Component>);
const ProjectTemplate: Story<Props> = (args) => (<Component {...args}>
    <ProjectView session={{}} />
</Component>);
const RoutineTemplate: Story<Props> = (args) => (<Component {...args}>
    <RoutineView session={{}} />
</Component>);
const StandardTemplate: Story<Props> = (args) => (<Component {...args}>
    <StandardView session={{}} />
</Component>);

// Export stories
export const Empty = EmptyTemplate.bind({});
export const Actor = ActorTemplate.bind({});
export const Organization = OrganizationTemplate.bind({});
export const Project = ProjectTemplate.bind({});
export const Routine = RoutineTemplate.bind({});
export const Standard = StandardTemplate.bind({});