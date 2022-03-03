import { Meta, Story } from "@storybook/react";
import { NodeColumn as Component } from '..';
import { NodeColumnProps as Props } from '../types';

// Define story metadata
export default {
    title: 'nodes/NodeColumn',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});