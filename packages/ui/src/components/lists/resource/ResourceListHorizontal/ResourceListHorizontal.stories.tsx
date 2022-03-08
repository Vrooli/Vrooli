import { Meta, Story } from "@storybook/react";
import { ResourceListHorizontal as Component } from '../..';
import { ResourceListHorizontalProps as Props } from '../types';

// Define story metadata
export default {
    title: 'lists/ResourceListHorizontal',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});