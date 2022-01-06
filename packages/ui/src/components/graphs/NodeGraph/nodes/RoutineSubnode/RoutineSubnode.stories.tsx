import { Meta, Story } from "@storybook/react";
import { RoutineSubnode as Component } from '..';
import { RoutineSubnodeProps as Props } from '../types';

// Define story metadata
export default {
    title: 'nodes/RoutineSubnode',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});