import { Meta, Story } from "@storybook/react";
import { UserView, OrganizationView, ProjectView, RoutineView, StandardView } from "components";
import { BaseObjectDialog as Component } from '..';
import { BaseObjectDialogProps as Props } from '../types';

// Define story metadata
export default {
    title: 'dialogs/BaseObjectDialog',
    component: Component,
} as Meta;

// Define templates for enabling control over props
const EmptyTemplate: Story<Props> = (args) => <Component {...args} />;
const OrganizationTemplate: Story<Props> = (args) => (<Component {...args}>
    <OrganizationView session={{}} zIndex={1} />
</Component>);
const ProjectTemplate: Story<Props> = (args) => (<Component {...args}>
    <ProjectView session={{}} zIndex={1} />
</Component>);
const RoutineTemplate: Story<Props> = (args) => (<Component {...args}>
    <RoutineView session={{}} zIndex={1} />
</Component>);
const StandardTemplate: Story<Props> = (args) => (<Component {...args}>
    <StandardView session={{}} zIndex={1} />
</Component>);
const UserTemplate: Story<Props> = (args) => (<Component {...args}>
    <UserView session={{}} zIndex={1} />
</Component>);

// Export stories
export const Empty = EmptyTemplate.bind({});
export const Organization = OrganizationTemplate.bind({});
export const Project = ProjectTemplate.bind({});
export const Routine = RoutineTemplate.bind({});
export const Standard = StandardTemplate.bind({});
export const User = UserTemplate.bind({});