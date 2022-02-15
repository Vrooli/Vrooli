import { Meta, Story } from "@storybook/react";
import { RoutineListNode as Component } from '..';
import { RoutineListNodeProps as Props } from '../types';

// Define story metadata
export default {
    title: 'nodes/RoutineListNode',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});